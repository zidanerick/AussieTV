package main

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestMPEGTSFFmpegArgumentsOnlyCopyAudioVideo(t *testing.T) {
	args := mpegTSFFmpegArgs("http://127.0.0.1:45782/jellyfin/hls/abc.m3u8?key=dummy")
	joined := " " + strings.Join(args, " ") + " "
	for _, required := range []string{" -c:v copy ", " -c:a copy ", " -f mpegts ", " -map 0:v:0 ", " -map 0:a:0 ", " -nostdin "} {
		if !strings.Contains(joined, required) {
			t.Fatalf("missing copy-only flag %q: %s", required, joined)
		}
	}
	for _, unsafe := range []string{" libx264 ", " -c:v h264 ", " -c:a aac ", " -re "} {
		if strings.Contains(joined, unsafe) {
			t.Fatalf("unexpected re-encode/realtime flag %q", unsafe)
		}
	}
}

func TestMPEGTSInternalRelayAuthenticationAndIsolation(t *testing.T) {
	cfg := exampleCfg()
	cfg.StreamMode = "mpegts"
	cfg.Bind = "0.0.0.0:45782"
	if localJellyfinURL(cfg) != "http://127.0.0.1:45782" {
		t.Fatal("internal upstream must remain loopback")
	}
	id := base64.RawURLEncoding.EncodeToString([]byte(sampleFeed().Channels[0].ID))
	h := newJellyfinHandler(cfg, func(string) ChannelData { return sampleFeed() })
	if r := requestJellyfin(t, h, "/jellyfin/hls/"+id+".m3u8", "GET"); r.Code != http.StatusUnauthorized {
		t.Fatalf("HLS bypassed auth: %d", r.Code)
	}
	if r := requestJellyfin(t, h, "/jellyfin/media/"+id+"?u=https://evil.example.org/x.ts&key="+url.QueryEscape(cfg.Key), "GET"); r.Code != http.StatusForbidden {
		t.Fatalf("media URL was not restricted: %d", r.Code)
	}
	if r := requestJellyfin(t, h, "/jellyfin/stream/"+id+".ts?key="+url.QueryEscape(cfg.Key), "HEAD"); r.Code != http.StatusOK || r.Header().Get("Content-Type") != "video/mp2t" {
		t.Fatalf("MPEG-TS probe: %d %v", r.Code, r.Header())
	}
	cfg.StreamMode = "redirect"
	h = newJellyfinHandler(cfg, func(string) ChannelData { return sampleFeed() })
	if r := requestJellyfin(t, h, "/jellyfin/hls/"+id+".m3u8?key="+url.QueryEscape(cfg.Key), "GET"); r.Code != http.StatusNotFound {
		t.Fatalf("internal relay enabled in default mode: %d", r.Code)
	}
}

func TestFFmpegFailureCategoriesAreSafe(t *testing.T) {
	cases := []struct{ message, want string }{
		{"https://token.secret.example/live?key=private: Server returned 403 Forbidden", "mpegts-source-forbidden"},
		{"http://127.0.0.1:45782/jellyfin/hls/abc: Connection refused", "mpegts-loopback-connection"},
		{"Invalid data found when processing input", "mpegts-invalid-source-media"},
		{"Server returned 401 Unauthorized", "mpegts-feed-authentication"},
		{"silent failure", "mpegts-no-transport-packets"},
		{"Stream map '0:a:0' matches no streams.", "mpegts-audio-track-not-found"},
	}
	for _, c := range cases {
		if got := classifyFFmpegFailure(c.message); got != c.want {
			t.Fatalf("category for %q = %q, want %q", c.message, got, c.want)
		}
	}
}

func TestFFmpegTailBounded(t *testing.T) {
	var tail ffmpegTail
	_, _ = tail.Write([]byte(strings.Repeat("x", 20000)))
	if len(tail.output()) != 8192 {
		t.Fatalf("stderr bounded to 8 KiB, got %d", len(tail.output()))
	}
}
