package main

// Jellyfin's built-in M3U/XMLTV integration is preferable to pretending to be
// an HDHomeRun device. This server exposes only read-only IPTV feed endpoints.
// It is intentionally separate from the desktop HTTP server and its quit API.

import (
	"aussietv/teletext"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type jellyfinConfig struct {
	Bind       string
	BaseURL    string
	Key        string
	Region     string
	Discovery  bool
	StreamMode string
}

func jellyfinConfigFromEnv() (jellyfinConfig, bool, error) {
	bind := strings.TrimSpace(os.Getenv("AUSSIETV_JELLYFIN_BIND"))
	if bind == "" {
		return jellyfinConfig{}, false, nil
	}
	cfg := jellyfinConfig{Bind: bind, BaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("AUSSIETV_JELLYFIN_BASE_URL")), "/"), Key: strings.TrimSpace(os.Getenv("AUSSIETV_JELLYFIN_KEY")), Region: safeRegion(strings.TrimSpace(os.Getenv("AUSSIETV_JELLYFIN_REGION"))), Discovery: os.Getenv("AUSSIETV_HDHR_DISCOVERY") == "1", StreamMode: strings.ToLower(strings.TrimSpace(os.Getenv("AUSSIETV_JELLYFIN_STREAM_MODE")))}
	if !validStreamMode(cfg.StreamMode) {
		return cfg, true, errors.New("AUSSIETV_JELLYFIN_STREAM_MODE must be redirect, relay, or mpegts")
	}
	if cfg.Discovery && !strings.HasPrefix(cfg.Bind, "0.0.0.0:") {
		return cfg, true, errors.New("HDHomeRun discovery requires a LAN Jellyfin bind")
	}
	if len(cfg.Key) < 24 {
		return cfg, true, errors.New("AUSSIETV_JELLYFIN_KEY must contain at least 24 characters")
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return cfg, true, errors.New("AUSSIETV_JELLYFIN_BASE_URL must be an http(s) address without path, credentials or query")
	}
	if _, err := url.Parse("http://" + cfg.Bind); err != nil {
		return cfg, true, fmt.Errorf("invalid AUSSIETV_JELLYFIN_BIND: %w", err)
	}
	return cfg, true, nil
}

func validStreamMode(s string) bool {
	return s == "" || s == "redirect" || s == "relay" || s == "mpegts"
}
func effectiveStreamMode(s string) string {
	if s == "relay" || s == "mpegts" {
		return s
	}
	return "redirect"
}

func xmlTVID(region string, ch Channel) string {
	return "aussietv." + strings.ToLower(region) + "." + base64.RawURLEncoding.EncodeToString([]byte(ch.ID))
}
func feedChannels(data ChannelData) []Channel {
	var channels []Channel
	for _, ch := range data.Channels {
		if ch.ID == "" || ch.Stream == "" {
			continue
		}
		if _, err := validateRemote(ch.Stream); err != nil {
			continue
		}
		channels = append(channels, ch)
	}
	return channels
}
func singleLine(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == '"' || r < 32 || r == 127 {
			return ' '
		}
		return r
	}, value)
}
func xmltvTime(sec int64) string { return time.Unix(sec, 0).UTC().Format("20060102150405 -0000") }
func m3uFeed(channels []Channel, cfg jellyfinConfig) string {
	var w strings.Builder
	w.WriteString("#EXTM3U\n")
	for _, ch := range channels {
		id := xmlTVID(cfg.Region, ch)
		logo := ""
		if _, err := validateRemote(ch.Logo); err == nil {
			logo = singleLine(ch.Logo)
		}
		fmt.Fprintf(&w, "#EXTINF:-1 tvg-id=\"%s\" tvg-name=\"%s\" tvg-chno=\"%s\" tvg-logo=\"%s\" group-title=\"%s\",%s\n",
			singleLine(id), singleLine(ch.Name), singleLine(ch.Number), logo, singleLine(ch.Network), singleLine(ch.Name))
		ext := ".m3u8"
		if effectiveStreamMode(cfg.StreamMode) == "mpegts" {
			ext = ".ts"
		}
		fmt.Fprintf(&w, "%s/jellyfin/stream/%s%s?key=%s\n", cfg.BaseURL, base64.RawURLEncoding.EncodeToString([]byte(ch.ID)), ext, url.QueryEscape(cfg.Key))
	}
	if teletextService != nil && teletext.HasVideoEncoder() {
		fmt.Fprintf(&w, "#EXTINF:-1 tvg-id=\"aussietv.teletext\" tvg-name=\"AussieTV Text\" tvg-chno=\"995\" group-title=\"Information\",AussieTV Text\n%s/jellyfin/teletext.ts?key=%s\n", cfg.BaseURL, url.QueryEscape(cfg.Key))
	}
	return w.String()
}

type xmltvName struct {
	Lang  string `xml:"lang,attr,omitempty"`
	Value string `xml:",chardata"`
}
type xmltvIcon struct {
	Src string `xml:"src,attr"`
}
type xmltvChannel struct {
	ID    string      `xml:"id,attr"`
	Names []xmltvName `xml:"display-name"`
	Icon  *xmltvIcon  `xml:"icon,omitempty"`
}
type xmltvProgramme struct {
	Start   string     `xml:"start,attr"`
	Stop    string     `xml:"stop,attr"`
	Channel string     `xml:"channel,attr"`
	Title   xmltvName  `xml:"title"`
	Icon    *xmltvIcon `xml:"icon,omitempty"`
}
type xmltvDocument struct {
	XMLName    xml.Name         `xml:"tv"`
	Generator  string           `xml:"generator-info-name,attr"`
	Channels   []xmltvChannel   `xml:"channel"`
	Programmes []xmltvProgramme `xml:"programme"`
}

func xmltvFeed(channels []Channel, region string) ([]byte, error) {
	doc := xmltvDocument{Generator: "AussieTV", Channels: []xmltvChannel{}, Programmes: []xmltvProgramme{}}
	for _, ch := range channels {
		id := xmlTVID(region, ch)
		row := xmltvChannel{ID: id, Names: []xmltvName{{Value: ch.Name}, {Value: ch.Number}}}
		if _, err := validateRemote(ch.Logo); err == nil {
			row.Icon = &xmltvIcon{Src: ch.Logo}
		}
		doc.Channels = append(doc.Channels, row)
		for _, p := range ch.Programs {
			if p.Start <= 0 || p.End <= p.Start || p.Title == "" {
				continue
			}
			row := xmltvProgramme{Start: xmltvTime(p.Start), Stop: xmltvTime(p.End), Channel: id, Title: xmltvName{Lang: "en", Value: p.Title}}
			if image, found := cachedProgrammeArtwork(p.Title); found && image != "" {
				row.Icon = &xmltvIcon{Src: image}
			}
			doc.Programmes = append(doc.Programmes, row)
		}
	}
	if teletextService != nil && teletext.HasVideoEncoder() {
		doc.Channels = append(doc.Channels, xmltvChannel{ID: "aussietv.teletext", Names: []xmltvName{{Value: "AussieTV Text"}, {Value: "995"}}})
		now := time.Now()
		doc.Programmes = append(doc.Programmes, xmltvProgramme{Start: xmltvTime(now.Add(-time.Hour).Unix()), Stop: xmltvTime(now.Add(24 * time.Hour).Unix()), Channel: "aussietv.teletext", Title: xmltvName{Lang: "en", Value: "AussieTV Text Information"}})
	}
	bytes, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), append(bytes, '\n')...), nil
}

// newJellyfinHandler has its own mux with no desktop APIs. It never accepts
// arbitrary URLs to fetch, preventing it from becoming a public proxy.
func newJellyfinHandler(cfg jellyfinConfig, load func(string) ChannelData) http.Handler {
	mux := http.NewServeMux()
	if teletextService != nil {
		mux.Handle("GET /teletext/", teletextService.Handler())
		mux.HandleFunc("GET /jellyfin/teletext.ts", func(w http.ResponseWriter, r *http.Request) {
			if err := teletextService.ServeMPEGTS(w, r); err != nil {
				log.Printf("Teletext video stream ended: %v", err)
			}
		})
	}
	hdhr := hdhrHTTP(cfg, load)
	mux.HandleFunc("GET /jellyfin/channels.m3u", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/x-mpegurl; charset=utf-8")
		_, _ = io.WriteString(w, m3uFeed(feedChannels(load(cfg.Region)), cfg))
	})
	mux.HandleFunc("GET /jellyfin/guide.xml", func(w http.ResponseWriter, r *http.Request) {
		channels := feedChannels(load(cfg.Region))
		content, err := xmltvFeed(channels, cfg.Region)
		if err != nil {
			http.Error(w, "guide encoding error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write(content)
		// Queue only after the HTTP response is produced. This does not block
		// XMLTV export or change existing channel IDs and channel logos.
		scheduleProgrammeArtwork(channels)
	})
	mux.HandleFunc("GET /jellyfin/stream/{id}", func(w http.ResponseWriter, r *http.Request) {
		idText := strings.TrimSuffix(strings.TrimSuffix(r.PathValue("id"), ".m3u8"), ".ts")
		id, err := base64.RawURLEncoding.DecodeString(idText)
		if err != nil || len(id) > 512 {
			http.NotFound(w, r)
			return
		}
		for _, ch := range feedChannels(load(cfg.Region)) {
			if ch.ID == string(id) {
				started := time.Now()
				// Jellyfin may probe sources using HEAD before it opens a channel.
				// Do not forward that probe or redirect it to an HLS origin:
				// some origins hold HEAD requests until Jellyfin's timeout.
				if effectiveStreamMode(cfg.StreamMode) == "mpegts" {
					w.Header().Set("Content-Type", "video/mp2t")
				} else {
					w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
				}
				if r.Method == http.MethodHead {
					w.WriteHeader(http.StatusOK)
					recordTune(ch.ID, "head", http.StatusOK, time.Since(started), 0)
					return
				}
				if effectiveStreamMode(cfg.StreamMode) == "mpegts" {
					serveJellyfinMPEGTS(w, r, cfg, ch)
					return
				}
				if effectiveStreamMode(cfg.StreamMode) == "relay" {
					relayPlaylistOrSegment(w, r, cfg, ch, ch.Stream)
					return
				}
				// Preserve direct broadcaster redirects for existing installations.
				// The redirect response is now measured too: previously events[]
				// stayed empty for every successful Direct-mode tune.
				http.Redirect(w, r, ch.Stream, http.StatusTemporaryRedirect)
				recordTune(ch.ID, "redirect", http.StatusTemporaryRedirect, time.Since(started), 0)
				return
			}
		}
		http.NotFound(w, r)
	})
	// Only the copy-only MPEG-TS process uses this root playlist. Its nested
	// segment URLs are rewritten through the protected media relay.
	mux.HandleFunc("GET /jellyfin/hls/{id}", func(w http.ResponseWriter, r *http.Request) {
		if effectiveStreamMode(cfg.StreamMode) != "mpegts" || !strings.HasSuffix(r.PathValue("id"), ".m3u8") {
			http.NotFound(w, r)
			return
		}
		text := strings.TrimSuffix(r.PathValue("id"), ".m3u8")
		id, err := base64.RawURLEncoding.DecodeString(text)
		if err != nil || len(id) > 512 {
			http.NotFound(w, r)
			return
		}
		for _, ch := range feedChannels(load(cfg.Region)) {
			if ch.ID == string(id) {
				local := cfg
				local.BaseURL = localJellyfinURL(cfg)
				relayPlaylistOrSegment(w, r, local, ch, ch.Stream)
				return
			}
		}
		http.NotFound(w, r)
	})
	mediaHandler := func(w http.ResponseWriter, r *http.Request) {
		id, err := base64.RawURLEncoding.DecodeString(r.PathValue("id"))
		if err != nil || len(id) == 0 || len(id) > 512 || (effectiveStreamMode(cfg.StreamMode) != "relay" && effectiveStreamMode(cfg.StreamMode) != "mpegts") {
			http.NotFound(w, r)
			return
		}
		remote := r.URL.Query().Get("u")
		for _, ch := range feedChannels(load(cfg.Region)) {
			if ch.ID != string(id) {
				continue
			}
			if _, err := validateRemote(remote); err != nil || !permittedStream(ch, remote, r.URL.Query().Get("sig")) {
				http.Error(w, "Unauthorised media resource", http.StatusForbidden)
				return
			}
			local := cfg
			if effectiveStreamMode(cfg.StreamMode) == "mpegts" {
				local.BaseURL = localJellyfinURL(cfg)
			}
			relayPlaylistOrSegment(w, r, local, ch, remote)
			return
		}
		http.NotFound(w, r)
	}
	// Backwards-compatible signed URLs plus new extension-bearing paths for
	// FFmpeg 9+ HLS format/extension validation. Both are authenticated above.
	mux.HandleFunc("GET /jellyfin/media/{id}", mediaHandler)
	mux.HandleFunc("GET /jellyfin/media/{id}/{name}", mediaHandler)
	mux.HandleFunc("GET /jellyfin/diagnostics", func(w http.ResponseWriter, r *http.Request) {
		relayDiagnostics(w, r, cfg)
	})
	mux.HandleFunc("GET /jellyfin/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "ok\n")
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Optional local-network HDHR compatibility endpoints deliberately
		// use LAN access instead of query-string credentials. They expose
		// only discovery, lineup and direct channel redirects.
		path := r.URL.Path
		if cfg.Discovery && (path == "/discover.json" || path == "/lineup.json" || path == "/lineup_status.json" || strings.HasPrefix(path, "/auto/")) {
			hdhr.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Referrer-Policy", "no-referrer")
		// A key must be present even for health checks; this is a LAN service.
		provided := sha256.Sum256([]byte(r.URL.Query().Get("key")))
		expected := sha256.Sum256([]byte(cfg.Key))
		if subtle.ConstantTimeCompare(provided[:], expected[:]) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		mux.ServeHTTP(w, r)
	})
}
