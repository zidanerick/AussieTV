package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type mpegTSFixtureTransport struct{ segment []byte }

func (t *mpegTSFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var content []byte
	mime := "application/vnd.apple.mpegurl"
	switch r.URL.Path {
	case "/live/test.m3u8":
		content = []byte("#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:3\n#EXT-X-MEDIA-SEQUENCE:0\n#EXTINF:3.0,\nseg.ts\n#EXT-X-ENDLIST\n")
	case "/live/seg.ts":
		content = t.segment
		mime = "video/mp2t"
	default:
		return &http.Response{StatusCode: 404, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("not found")), Request: r}, nil
	}
	hdr := make(http.Header)
	hdr.Set("Content-Type", mime)
	return &http.Response{StatusCode: 200, Header: hdr, Body: io.NopCloser(bytes.NewReader(content)), ContentLength: int64(len(content)), Request: r}, nil
}

func TestMPEGTSRealStreamCopyAndDisconnect(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("FFmpeg unavailable in CI; opt-in relay requires separate installation")
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe unavailable")
	}
	// Produce a short H.264 + AAC fixture entirely offline.
	command := exec.Command(ffmpeg, "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=320x180:rate=25", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000", "-t", "3", "-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "64k", "-f", "mpegts", "pipe:1")
	fixture, err := command.Output()
	if err != nil {
		t.Skipf("FFmpeg cannot generate the offline H.264/AAC fixture: %v", err)
	}
	old := mediaClient
	mediaClient = &http.Client{Transport: &mpegTSFixtureTransport{segment: fixture}, Timeout: 10 * time.Second}
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
	ch.Stream = "https://streams.example.org/live/test.m3u8"
	server := &http.Server{Handler: newJellyfinHandler(cfg, func(string) ChannelData { return ChannelData{Channels: []Channel{ch}} }), ReadHeaderTimeout: 3 * time.Second}
	go func() { _ = server.Serve(ln) }()
	t.Cleanup(func() { _ = server.Close() })
	id := base64.RawURLEncoding.EncodeToString([]byte(ch.ID))
	endpoint := fmt.Sprintf("%s/jellyfin/stream/%s.ts?key=%s", cfg.BaseURL, id, url.QueryEscape(cfg.Key))
	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("MPEGTS request: %v", err)
	}
	if response.StatusCode != 200 || response.Header.Get("Content-Type") != "video/mp2t" {
		response.Body.Close()
		t.Fatalf("TS HTTP response %d %v", response.StatusCode, response.Header)
	}
	first := make([]byte, 188)
	if _, err := io.ReadFull(response.Body, first); err != nil || first[0] != 0x47 {
		t.Fatalf("invalid TS first packet: %v %x", err, first[0])
	}
	// Client hangup must tear down the streaming FFmpeg child.
	_ = response.Body.Close()
	cancel()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) && activeMPEGTS.Load() != 0 {
		time.Sleep(20 * time.Millisecond)
	}
	if activeMPEGTS.Load() != 0 {
		t.Fatalf("MPEG-TS process survived client disconnect, active=%d", activeMPEGTS.Load())
	}
	// Re-open and retain the complete stream to inspect copied codecs.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 18*time.Second)
	defer cancel2()
	req2, _ := http.NewRequestWithContext(ctx2, http.MethodGet, endpoint, nil)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := io.ReadAll(io.LimitReader(resp2.Body, 8<<20))
	resp2.Body.Close()
	if err != nil {
		t.Fatalf("read remuxed MPEG-TS: %v", err)
	}
	if len(contents) < 188*10 || contents[0] != 0x47 {
		t.Fatalf("invalid TS output, length=%d", len(contents))
	}
	path := filepath.Join(t.TempDir(), "copy.ts")
	if err := os.WriteFile(path, contents, 0600); err != nil {
		t.Fatal(err)
	}
	inspect := exec.Command(ffprobe, "-v", "error", "-show_entries", "stream=codec_name", "-of", "csv=p=0", path)
	info, err := inspect.Output()
	if err != nil {
		t.Fatalf("ffprobe could not decode output: %v", err)
	}
	if !bytes.Contains(info, []byte("h264")) || !bytes.Contains(info, []byte("aac")) {
		t.Fatalf("codecs were not copied: %s", info)
	}
	// Check the first audio and video PTS in the offline remux, not just the
	// advertised codec names. This is not a substitute for listening on Jellyfin.
	firstPTS := func(selector string) float64 {
		probe := exec.Command(ffprobe, "-v", "error", "-select_streams", selector, "-show_entries", "stream=start_time", "-of", "default=noprint_wrappers=1:nokey=1", path)
		out, e := probe.Output()
		if e != nil {
			t.Fatalf("PTS probe %s: %v", selector, e)
		}
		first := strings.TrimSpace(strings.Split(string(out), "\n")[0])
		val, e := strconv.ParseFloat(first, 64)
		if e != nil {
			t.Fatalf("unreadable PTS %s: %q", selector, first)
		}
		return val
	}
	videoPTS, audioPTS := firstPTS("v:0"), firstPTS("a:0")
	if delta := math.Abs(videoPTS - audioPTS); delta > 0.25 {
		t.Fatalf("offline stream A/V start differs by %.3fs (video %.3f, audio %.3f)", delta, videoPTS, audioPTS)
	}
	t.Logf("offline remux audio/video start gap: %.3fs", math.Abs(videoPTS-audioPTS))
	t.Logf("offline copy-only remux: %d bytes, codecs %q; not a Jellyfin Web click-to-picture benchmark", len(contents), strings.TrimSpace(string(info)))
}
