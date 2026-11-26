package main

// A Jellyfin-only HLS relay. The remote, untrusted channel index is already
// constrained by feedChannels; HLS descendants additionally require a
// per-process HMAC, and every upstream request uses mediaClient's public-IP
// dial/redirect policy. No arbitrary-URL proxy endpoint is exposed.

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// A bounded history for diagnosing whether latency is in AussieTV or after
// Jellyfin has received the stream. No remote URLs, tokens or access keys.
type tuneEvent struct {
	At         string `json:"at"`
	Channel    string `json:"channel"`
	Mode       string `json:"mode"`
	Status     int    `json:"status"`
	TotalMS    int64  `json:"totalMs"`
	UpstreamMS int64  `json:"upstreamMs,omitempty"`
}

// Aggregate counters make successful media requests visible without emitting
// a log line for every HLS fragment. They do not contain credentials or URLs.
var relayManifestRequests atomic.Int64
var relayMediaRequests atomic.Int64
var relayMediaFailures atomic.Int64
var relayMediaBytes atomic.Int64

var tuneHistory = struct {
	sync.Mutex
	events []tuneEvent
}{}

func recordTune(channel, mode string, code int, total, upstream time.Duration) {
	event := tuneEvent{At: time.Now().UTC().Format(time.RFC3339), Channel: channel,
		Mode: mode, Status: code, TotalMS: total.Milliseconds(), UpstreamMS: upstream.Milliseconds()}
	tuneHistory.Lock()
	tuneHistory.events = append(tuneHistory.events, event)
	if len(tuneHistory.events) > 32 {
		tuneHistory.events = append([]tuneEvent(nil), tuneHistory.events[len(tuneHistory.events)-32:]...)
	}
	tuneHistory.Unlock()
	if mode != "segment" || code >= 400 {
		log.Printf("Jellyfin %s channel=%q status=%d total_ms=%d upstream_ms=%d", mode, channel, code, event.TotalMS, event.UpstreamMS)
	}
}

func tuneEvents() []tuneEvent {
	tuneHistory.Lock()
	defer tuneHistory.Unlock()
	result := make([]tuneEvent, len(tuneHistory.events))
	copy(result, tuneHistory.events)
	return result
}

// Rewriting all playlist URLs back through the authenticated listener keeps
// FFmpeg/Jellyfin's media requests on the same origin. This avoids the 307
// handoff; it is opt-in because some broadcaster CDNs require direct requests.
// FFmpeg's HLS demuxer validates media/segment URL extensions (including
// fMP4 init fragments). Older AussieTV relay links ended at /media/<id> and
// hid the real suffix in ?u=; modern FFmpeg can reject those URLs before
// attempting the segment download. The visible basename is only a safe hint;
// the HMAC still binds the full upstream URL to the indexed channel.
func relayMediaExtension(remote, fallback string) string {
	u, err := url.Parse(remote)
	if err == nil {
		ext := strings.ToLower(path.Ext(u.Path))
		switch ext {
		case ".m3u8", ".ts", ".m4s", ".mp4", ".m4a", ".aac", ".ac3", ".eac3", ".ec3", ".vtt", ".webvtt", ".cmfv", ".cmfa", ".fmp4", ".mp3", ".key", ".jpg", ".jpeg":
			return ext
		}
	}
	return fallback
}
func relayURLWithHint(cfg jellyfinConfig, chID, remote, fallback string) string {
	id := base64.RawURLEncoding.EncodeToString([]byte(chID))
	query := url.Values{}
	query.Set("key", cfg.Key)
	query.Set("u", remote)
	query.Set("sig", streamSignature(chID, remote))
	return cfg.BaseURL + "/jellyfin/media/" + id + "/resource" + relayMediaExtension(remote, fallback) + "?" + query.Encode()
}
func relayURL(cfg jellyfinConfig, chID, remote string) string {
	return relayURLWithHint(cfg, chID, remote, ".ts")
}

func rewriteRelayPlaylist(body []byte, base *url.URL, cfg jellyfinConfig, chID string) ([]byte, error) {
	if len(body) == 0 || len(body) > 5<<20 || !strings.HasPrefix(strings.TrimPrefix(string(body), "\ufeff"), "#EXTM3U") {
		return nil, errors.New("provider did not return a valid HLS playlist")
	}
	lines := strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n")
	for i, item := range lines {
		item = strings.TrimSpace(item)
		if strings.HasPrefix(item, "#EXT-X-KEY") || strings.HasPrefix(item, "#EXT-X-SESSION-KEY") {
			if strings.Contains(item, "KEYFORMAT=") || strings.Contains(item, "METHOD=SAMPLE-AES") {
				return nil, errors.New("protected HLS playlist is not supported")
			}
		}
		if item == "" {
			continue
		}
		if !strings.HasPrefix(item, "#") {
			target, err := validatedPlaylistRef(base, item)
			if err != nil {
				return nil, err
			}
			hint := ".ts"
			if i > 0 && strings.HasPrefix(strings.TrimSpace(lines[i-1]), "#EXT-X-STREAM-INF") {
				hint = ".m3u8"
			}
			lines[i] = relayURLWithHint(cfg, chID, target, hint)
		} else if strings.Contains(item, `URI="`) {
			var replaceErr error
			lines[i] = uriRegexp.ReplaceAllStringFunc(item, func(attr string) string {
				if replaceErr != nil {
					return attr
				}
				value := uriRegexp.FindStringSubmatch(attr)
				if len(value) < 2 {
					replaceErr = errors.New("invalid HLS URI attribute")
					return attr
				}
				target, err := validatedPlaylistRef(base, value[1])
				if err != nil {
					replaceErr = err
					return attr
				}
				hint := ".ts"
				switch {
				case strings.HasPrefix(item, "#EXT-X-MEDIA"), strings.HasPrefix(item, "#EXT-X-RENDITION-REPORT"):
					hint = ".m3u8"
				case strings.HasPrefix(item, "#EXT-X-MAP"):
					hint = ".mp4"
				case strings.HasPrefix(item, "#EXT-X-KEY"), strings.HasPrefix(item, "#EXT-X-SESSION-KEY"):
					hint = ".key"
				case strings.HasPrefix(item, "#EXT-X-PART"), strings.HasPrefix(item, "#EXT-X-PRELOAD-HINT"):
					hint = ".m4s"
				}
				return `URI="` + relayURLWithHint(cfg, chID, target, hint) + `"`
			})
			if replaceErr != nil {
				return nil, replaceErr
			}
		}
	}
	return []byte(strings.Join(lines, "\n")), nil
}
func validatedPlaylistRef(base *url.URL, raw string) (string, error) {
	relative, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	resolved := base.ResolveReference(relative)
	if _, err = validateRemote(resolved.String()); err != nil {
		return "", fmt.Errorf("blocked HLS subresource: %w", err)
	}
	return resolved.String(), nil
}

// relayPlaylistOrSegment fetches manifests or binary media from indexed roots
// and HMAC-signed descendants. The caller has authenticated the feed request.
func relayPlaylistOrSegment(w http.ResponseWriter, r *http.Request, cfg jellyfinConfig, ch Channel, remote string) {
	started := time.Now()
	mode := "segment"
	original, err := validateRemote(remote)
	if err != nil {
		http.Error(w, "Invalid media address", 400)
		recordTune(ch.ID, mode, 400, time.Since(started), 0)
		return
	}
	if strings.HasSuffix(strings.ToLower(original.Path), ".m3u8") {
		mode = "manifest"
	}
	if mode == "manifest" {
		relayManifestRequests.Add(1)
	}
	if r.Method == http.MethodHead {
		if mode == "manifest" {
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		} else {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, remote, nil)
	if err != nil {
		http.Error(w, "Invalid media request", 400)
		return
	}
	setProviderRequestHeaders(req, ch)
	if value := r.Header.Get("Range"); value != "" {
		req.Header.Set("Range", value)
	}
	// mediaClient enforces public DNS targets, public redirect targets and
	// finite upstream response-header timeouts.
	response, err := mediaClient.Do(req)
	upstream := time.Since(started)
	if err != nil {
		if mode != "manifest" {
			relayMediaFailures.Add(1)
		}
		http.Error(w, "Provider request unavailable", http.StatusBadGateway)
		recordTune(ch.ID, mode, 502, time.Since(started), upstream)
		return
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		if mode != "manifest" {
			relayMediaFailures.Add(1)
		}
		http.Error(w, fmt.Sprintf("Provider returned HTTP %d", response.StatusCode), http.StatusBadGateway)
		recordTune(ch.ID, mode, 502, time.Since(started), upstream)
		return
	}
	finalURL := response.Request.URL
	mime := strings.ToLower(response.Header.Get("Content-Type"))
	isPlaylist := strings.HasSuffix(strings.ToLower(finalURL.Path), ".m3u8") || strings.Contains(mime, "mpegurl") || strings.HasSuffix(strings.ToLower(original.Path), ".m3u8")
	if isPlaylist {
		if mode != "manifest" {
			relayManifestRequests.Add(1)
		}
		mode = "manifest"
		limited := io.LimitReader(response.Body, (5<<20)+1)
		body, err := io.ReadAll(limited)
		if err != nil || len(body) > 5<<20 {
			http.Error(w, "Manifest too large or unavailable", 502)
			recordTune(ch.ID, mode, 502, time.Since(started), upstream)
			return
		}
		transformed, err := rewriteRelayPlaylist(body, finalURL, cfg, ch.ID)
		if err != nil {
			http.Error(w, "Unsupported HLS playlist", 415)
			recordTune(ch.ID, mode, 415, time.Since(started), upstream)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(transformed)
		recordTune(ch.ID, mode, 200, time.Since(started), upstream)
		return
	}
	for _, header := range []string{"Content-Type", "Content-Range", "Accept-Ranges", "Content-Length"} {
		if value := response.Header.Get(header); value != "" {
			w.Header().Set(header, value)
		}
	}
	// Record successful upstream segment requests even if the client disconnects.
	relayMediaRequests.Add(1)
	// Do not retain HTTP Server's 45-second write timeout on a large or slow
	// media segment; cancellation is still tied to the incoming HTTP request.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	w.WriteHeader(response.StatusCode)
	transferred, copyErr := io.Copy(w, response.Body)
	relayMediaBytes.Add(transferred)
	err = copyErr
	if err != nil && r.Context().Err() == nil {
		log.Printf("Jellyfin segment transfer ended channel=%q: %v", ch.ID, err)
	}
	if err != nil {
		relayMediaFailures.Add(1)
		recordTune(ch.ID, mode, 499, time.Since(started), upstream)
	}
}

func relayDiagnostics(w http.ResponseWriter, _ *http.Request, cfg jellyfinConfig) {
	posters, misses, pending := artworkQueueStats()
	ffmpegPresent, ffmpegRunnable, ffmpegSource := ffmpegHealth()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"appVersion":                  version,
		"streamMode":                  effectiveStreamMode(cfg.StreamMode),
		"streamModeConfigured":        cfg.StreamMode,
		"mpegtsActiveStreams":         activeMPEGTS.Load(),
		"fastTuneSessionsStarted": fastTuneStarts.Load(),
		"fastTuneSessionReuses": fastTuneReuses.Load(),
		"fastTuneLagDrops": fastTuneLagDrops.Load(),
		"fastTuneIdleSeconds": int(fastTuneIdle.Seconds()),
		"fastTuneMaxChannels": fastTuneChannels,
		"fastTuneBufferBytesPerChannel": fastTunePackets*188,
		"ffmpegPresent":               ffmpegPresent,
		"ffmpegRunnable":              ffmpegRunnable,
		"ffmpegSource":                ffmpegSource,
		"mpegtsStartupTimeoutSeconds": int(mpegTSStartupTimeout.Seconds()),
		"programmeArtworkCached":      posters,
		"programmeArtworkMisses":      misses,
		"programmeArtworkPending":     pending,
		"events":                      tuneEvents(),
		"relayManifestRequests":       relayManifestRequests.Load(),
		"relayMediaRequests":          relayMediaRequests.Load(),
		"relayMediaFailures":          relayMediaFailures.Load(),
		"relayMediaBytes":             relayMediaBytes.Load(),
		"hdhomerunEnabled":            cfg.Discovery,
		"hdhomerunUdpListening":       discoveryUDPListening.Load(),
		"hdhomerunHttpListening":      discoveryHTTPListening.Load(),
		"hdhomerunUdpProbesReceived":  discoveryProbes.Load(),
		"hdhomerunRepliesSent":        discoveryReplies.Load(),
		"hdhomerunHttpRequests":       discoveryHTTPRequests.Load(),
		"note":                        "Direct mode now records HTTP HEAD and redirect responses. Fast redirects cannot measure the later broadcaster/Jellyfin buffering time. Discovery enabled does not imply the UDP and HTTP listeners are active.",
	})
}
