package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fmp4FixtureTransport struct {
	files map[string][]byte
}

func (f *fmp4FixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	name := filepath.Base(r.URL.Path)
	data, ok := f.files[name]
	if r.URL.Path == "/live/master.m3u8" {
		data = []byte("#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=500000\ntest.m3u8\n")
		ok = true
	}
	status := 200
	if !ok {
		status = 404
		data = []byte("missing")
	}
	mime := "application/octet-stream"
	switch filepath.Ext(name) {
	case ".m3u8":
		mime = "application/vnd.apple.mpegurl"
	case ".mp4", ".m4s":
		mime = "video/mp4"
	}
	h := make(http.Header)
	h.Set("Content-Type", mime)
	return &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(bytes.NewReader(data)), ContentLength: int64(len(data)), Request: r}, nil
}
func TestMPEGTSFFmpegFMP4SourceWithExtensionBearingRelay(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("FFmpeg unavailable")
	}
	dir := t.TempDir()
	// Exercise an actual fMP4 init map, m4s segments and variant playlist.
	output := filepath.Join(dir, "test.m3u8")
	makeCmd := exec.Command(ffmpeg, "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=240x136:rate=20", "-f", "lavfi", "-i", "sine=frequency=330:sample_rate=48000", "-t", "3", "-c:v", "libx264", "-preset", "ultrafast", "-g", "20", "-c:a", "aac", "-b:a", "64k", "-hls_segment_type", "fmp4", "-hls_time", "1", "-hls_list_size", "0", "-hls_playlist_type", "vod", "-f", "hls", output)
	if output, err := makeCmd.CombinedOutput(); err != nil {
		t.Skipf("FFmpeg fixture generator unavailable: %v %s", err, output)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	fake := &fmp4FixtureTransport{files: make(map[string][]byte)}
	for _, entry := range entries {
		b, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		fake.files[entry.Name()] = b
	}
	old := mediaClient
	mediaClient = &http.Client{Transport: fake, Timeout: 5 * time.Second}
	t.Cleanup(func() { mediaClient = old })
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cfg := exampleCfg()
	cfg.StreamMode = "mpegts"
	cfg.Bind = ln.Addr().String()
	cfg.BaseURL = "http://" + ln.Addr().String()
	ch := sampleFeed().Channels[0]
	ch.Stream = "https://streams.example.org/live/master.m3u8"
	srv := &http.Server{Handler: newJellyfinHandler(cfg, func(string) ChannelData { return ChannelData{Channels: []Channel{ch}} }), ReadHeaderTimeout: 3 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	id := base64.RawURLEncoding.EncodeToString([]byte(ch.ID))
	endpoint := fmt.Sprintf("%s/jellyfin/stream/%s.ts?key=%s", cfg.BaseURL, id, url.QueryEscape(cfg.Key))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("fMP4 relay request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		t.Fatalf("fMP4 relay %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 188*5 || b[0] != 0x47 {
		t.Fatalf("invalid MPEG-TS output %d", len(b))
	}
	t.Logf("FFmpeg 7+ HLS demuxer accepted extension-bearing fMP4 relay, %d bytes", len(b))
}
