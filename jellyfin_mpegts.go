package main

// Copy-only MPEG-TS prototype for Jellyfin. FFmpeg reads an authenticated,
// local HLS relay, not arbitrary broadcaster URLs. All remote manifests,
// variant playlists and segments pass through mediaClient's SSRF guards.
// This mode is opt-in and intentionally does not modify the default redirect.

import (
	"aussietv/internal/ffmpegexec"
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Visible only to authenticated diagnostics; helps confirm stopped streams
// really release their FFmpeg process after Jellyfin disconnects.
var activeMPEGTS atomic.Int64

// A finite startup bound prevents silent, indefinitely pending requests if
// a broadcaster HLS manifest fails, FFmpeg cannot open the loopback relay or
// the transport stream never emits its first packet. Live transfer after the
// first packet is not timed out by this bound.
var mpegTSStartupTimeout = 18 * time.Second

// Only sanitized error categories appear in authenticated diagnostics. FFmpeg
// stderr often contains broadcaster-signed URLs and the local feed key.
type ffmpegTail struct {
	mu   sync.Mutex
	data []byte
}

func (t *ffmpegTail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data = append(t.data, p...)
	if len(t.data) > 8192 {
		t.data = append([]byte(nil), t.data[len(t.data)-8192:]...)
	}
	return len(p), nil
}
func (t *ffmpegTail) output() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(bytes.Clone(t.data))
}

func classifyFFmpegFailure(message string) string {
	x := strings.ToLower(message)
	switch {
	case strings.Contains(x, "matches no streams") && strings.Contains(x, "0:a:0"):
		return "mpegts-audio-track-not-found"
	case strings.Contains(x, "connection refused") || strings.Contains(x, "cannot connect"):
		return "mpegts-loopback-connection"
	case strings.Contains(x, " 401") || strings.Contains(x, "error 401") || strings.Contains(x, "unauthorized"):
		return "mpegts-feed-authentication"
	case strings.Contains(x, " 403") || strings.Contains(x, "error 403") || strings.Contains(x, "forbidden"):
		return "mpegts-source-forbidden"
	case strings.Contains(x, " 404") || strings.Contains(x, "error 404") || strings.Contains(x, "not found"):
		return "mpegts-source-not-found"
	case strings.Contains(x, " 415") || strings.Contains(x, "unsupported hls playlist"):
		return "mpegts-protected-or-unsupported-hls"
	case strings.Contains(x, "allowed_extensions") || strings.Contains(x, "allowed_segment_extensions") || strings.Contains(x, "extension mismatches") || strings.Contains(x, "not a common multimedia extension"):
		return "mpegts-media-extension-rejected"
	case strings.Contains(x, "error opening input file") || strings.Contains(x, "failed to open") || strings.Contains(x, "error when loading first segment"):
		return "mpegts-ffmpeg-hls-input-failed"
	case strings.Contains(x, "invalid data") || strings.Contains(x, "error parsing"):
		return "mpegts-invalid-source-media"
	case strings.Contains(x, "protocol not found") || strings.Contains(x, "protocol is not on whitelist"):
		return "mpegts-unsupported-protocol"
	case strings.Contains(x, "timed out") || strings.Contains(x, "timeout"):
		return "mpegts-upstream-timeout"
	case strings.Contains(x, "permission denied") || strings.Contains(x, "access is denied"):
		return "mpegts-permission-denied"
	case strings.Contains(x, "no streams") || strings.Contains(x, "output file #0 does not contain any stream"):
		return "mpegts-no-compatible-streams"
	default:
		return "mpegts-no-transport-packets"
	}
}

// ffmpegHealth reports if the bundled executable can actually start.
// A binary's presence alone does not prove Windows permits it to execute.
// Never expose absolute local paths, feed keys or raw command stderr.
func ffmpegHealth() (bool, bool, string) {
	executable, err := ffmpegexec.Find()
	if err != nil {
		return false, false, "missing"
	}
	source := "system"
	if osExe, e := os.Executable(); e == nil {
		dir := filepath.Dir(osExe)
		if executable == filepath.Join(dir, "ffmpeg", "ffmpeg.exe") || executable == filepath.Join(dir, "ffmpeg.exe") ||
			executable == filepath.Join(dir, "ffmpeg", "ffmpeg") || executable == filepath.Join(dir, "ffmpeg") {
			source = "bundled"
		}
	}
	if os.Getenv("AUSSIETV_FFMPEG_PATH") != "" {
		source = "override"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-version")
	ffmpegexec.Configure(cmd)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	err = cmd.Run()
	return true, err == nil, source
}

func localJellyfinURL(cfg jellyfinConfig) string {
	_, port, err := net.SplitHostPort(cfg.Bind)
	if err != nil {
		return ""
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return ""
	}
	return "http://127.0.0.1:" + port
}

func mpegTSFFmpegArgs(source string) []string {
	// Require one video and one audio track. The previous optional audio map
	// silently produced video-only television when an HLS audio rendition was
	// absent from FFmpeg's short input probe. A moderately larger probe budget
	// is a maximum, not a fixed startup delay; direct redirect stays default.
	// Preserve source codecs and audio timestamps without re-encoding.
	return []string{
		"-hide_banner", "-loglevel", "error", "-nostdin",
		"-rw_timeout", "12000000", "-analyzeduration", "3000000", "-probesize", "2097152",
		"-i", source,
		"-map", "0:v:0", "-map", "0:a:0", "-c:v", "copy", "-c:a", "copy",
		"-sn", "-dn", "-muxdelay", "0", "-muxpreload", "0",
		"-f", "mpegts", "pipe:1",
	}
}

func serveJellyfinMPEGTS(w http.ResponseWriter, r *http.Request, cfg jellyfinConfig, ch Channel) {
	start := time.Now()
	base := localJellyfinURL(cfg)
	ffmpeg, err := ffmpegexec.Find()
	if base == "" || err != nil {
		http.Error(w, "MPEG-TS requires bundled ffmpeg/ffmpeg.exe, an FFmpeg on PATH, and a fixed feed port", http.StatusServiceUnavailable)
		recordTune(ch.ID, "mpegts-unavailable", http.StatusServiceUnavailable, time.Since(start), 0)
		return
	}
	id := base64.RawURLEncoding.EncodeToString([]byte(ch.ID))
	input := base + "/jellyfin/hls/" + id + ".m3u8?key=" + url.QueryEscape(cfg.Key)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	args := mpegTSFFmpegArgs(input)
	if cfg.StreamMode == "fast-tune" {
		// Emit PAT/PMT at video random-access points for safe shared joins.
		args = append(args[:len(args)-2], append([]string{"-mpegts_flags","+resend_headers"},args[len(args)-2:]...)...)
	}
	command := exec.CommandContext(ctx, ffmpeg, args...)
	ffmpegexec.Configure(command)
	stdout, err := command.StdoutPipe()
	if err != nil {
		http.Error(w, "Cannot prepare MPEG-TS relay", 502)
		return
	}
	// Keep only a bounded local tail of FFmpeg's errors. Raw stderr must
	// NEVER be logged or returned, since it contains signed broadcaster URLs.
	stderr := &ffmpegTail{}
	command.Stderr = stderr
	if err := command.Start(); err != nil {
		http.Error(w, "FFmpeg failed to launch; see authenticated diagnostics", 502)
		recordTune(ch.ID, "mpegts-executable-start-failed", 502, time.Since(start), 0)
		return
	}
	activeMPEGTS.Add(1)
	defer activeMPEGTS.Add(-1)

	var startupExpired atomic.Bool
	startupTimer := time.AfterFunc(mpegTSStartupTimeout, func() {
		startupExpired.Store(true)
		cancel()
	})
	defer startupTimer.Stop()

	reader := bufio.NewReaderSize(stdout, 32*1024)
	// The former first-packet gate accepted a PAT or video packet, with no
	// assurance an audio track was present. Jellyfin then played a silent
	// picture indefinitely. Inspect the PAT/PMT and at least one audio packet
	// before sending HTTP 200. Keep the read-ahead small to avoid buffering
	// entire live segments or increasing normal tune latency substantially.
	var inspected tsAudioProbe
	var initial bytes.Buffer
	packet := make([]byte, 188)
	var readErr error
	var audioExpired atomic.Bool
	var audioTimer *time.Timer
	for initial.Len() < 1024*1024 {
		_, readErr = io.ReadFull(reader, packet)
		if readErr != nil || packet[0] != 0x47 {
			break
		}
		initial.Write(packet)
		inspected.add(packet)
		if audioTimer == nil {
			if cfg.StreamMode == "fast-tune" { recordTune(ch.ID,"fast-tune-first-packet",200,time.Since(start),0) }
			audioTimer = time.AfterFunc(5*time.Second, func() { audioExpired.Store(true); cancel() })
		}
		if inspected.audioReady() {
			break
		}
	}
	startupTimer.Stop()
	if audioTimer != nil {
		audioTimer.Stop()
	}
	if readErr != nil || !inspected.audioReady() {
		cancel()
		_ = command.Wait()
		if r.Context().Err() != nil {
			return
		}
		category := classifyFFmpegFailure(stderr.output())
		code := http.StatusBadGateway
		if inspected.hasVideo && !inspected.hasAudio {
			category = "mpegts-audio-track-missing"
		} else if inspected.hasAudio && !inspected.audioPacketSeen {
			category = "mpegts-audio-packets-missing"
		} else if initial.Len() > 0 && !inspected.hasVideo {
			category = "mpegts-video-track-missing"
		}
		if startupExpired.Load() || audioExpired.Load() {
			if initial.Len() == 0 {
				category = "mpegts-startup-timeout"
			}
			code = http.StatusGatewayTimeout
		}
		http.Error(w, "MPEG-TS stream unavailable ("+category+"); check the authenticated diagnostics", code)
		recordTune(ch.ID, category, code, time.Since(start), 0)
		return
	}
	// A live channel must not inherit the feed server's finite WriteTimeout.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(initial.Bytes()); err == nil {
		_ = http.NewResponseController(w).Flush()
		recordTune(ch.ID, "mpegts-audio-video-ready", 200, time.Since(start), 0)
		_, err = io.Copy(w, reader)
	}
	// A disconnected HTTP client cancels r.Context(), which terminates FFmpeg.
	// On a normal finite HLS fixture EOF, allow FFmpeg to exit normally.
	if err != nil || r.Context().Err() != nil {
		cancel()
	}
	waitErr := command.Wait()
	if err != nil && !errors.Is(err, context.Canceled) && r.Context().Err() == nil {
		log.Printf("Jellyfin MPEG-TS channel=%q delivery ended: %T", ch.ID, err)
	}
	if waitErr != nil && r.Context().Err() == nil && err == nil {
		category := classifyFFmpegFailure(stderr.output())
		recordTune(ch.ID, category, 502, time.Since(start), 0)
		log.Printf("Jellyfin MPEG-TS channel=%q ended: %s", ch.ID, category)
	}
}

// The FFmpeg input is always a local authenticated URL. The only network
// requests outside the computer are resolved by the guarded HLS relay.
func checkMPEGTSInput(cfg jellyfinConfig) error {
	if localJellyfinURL(cfg) == "" {
		return fmt.Errorf("MPEG-TS relay needs a fixed local listener")
	}
	if strings.TrimSpace(cfg.Key) == "" {
		return fmt.Errorf("MPEG-TS relay requires feed authentication")
	}
	return nil
}
