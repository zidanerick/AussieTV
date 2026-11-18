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
	"os/exec"
	"strings"
	"testing"
	"time"
)

func makeTestTSPackets(t *testing.T, audio bool) []byte {
	t.Helper()
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("FFmpeg needed for offline stream probe test")
	}
	args := []string{"-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=160x90:rate=25"}
	if audio {
		args = append(args, "-f", "lavfi", "-i", "sine=frequency=660:sample_rate=48000")
	}
	args = append(args, "-t", "2", "-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p")
	if audio {
		args = append(args, "-c:a", "aac", "-b:a", "64k")
	} else {
		args = append(args, "-an")
	}
	args = append(args, "-f", "mpegts", "pipe:1")
	output, err := exec.Command(ffmpeg, args...).Output()
	if err != nil {
		t.Skipf("cannot create offline TS fixture: %v", err)
	}
	return output
}

func TestTSInspectorRequiresAudioAndVideoPackets(t *testing.T) {
	for _, tc := range []struct {
		name  string
		audio bool
	}{{"with AAC", true}, {"no audio", false}} {
		t.Run(tc.name, func(t *testing.T) {
			data := makeTestTSPackets(t, tc.audio)
			var probe tsAudioProbe
			for i := 0; i+188 <= len(data); i += 188 {
				probe.add(data[i : i+188])
			}
			if !probe.hasVideo || !probe.videoPacketSeen {
				t.Fatal("failed to find video PID and packets")
			}
			if probe.hasAudio != tc.audio || probe.audioPacketSeen != tc.audio || probe.audioReady() != tc.audio {
				t.Fatalf("audio=%v probe=%+v", tc.audio, probe)
			}
		})
	}
}

func TestMPEGTSVideoOnlyFailsBeforeSendingHTTP200(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("FFmpeg not installed")
	}
	_ = ffmpeg
	segment := makeTestTSPackets(t, false)
	previous := mediaClient
	mediaClient = &http.Client{Transport: &mpegTSFixtureTransport{segment: segment}, Timeout: 10 * time.Second}
	t.Cleanup(func() { mediaClient = previous })
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cfg := exampleCfg()
	cfg.StreamMode = "mpegts"
	cfg.Bind = listener.Addr().String()
	cfg.BaseURL = "http://" + cfg.Bind
	ch := sampleFeed().Channels[0]
	ch.Stream = "https://streams.example.org/live/test.m3u8"
	server := &http.Server{Handler: newJellyfinHandler(cfg, func(string) ChannelData { return ChannelData{Channels: []Channel{ch}} }), ReadHeaderTimeout: 3 * time.Second}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	chID := base64.RawURLEncoding.EncodeToString([]byte(ch.ID))
	target := fmt.Sprintf("%s/jellyfin/stream/%s.ts?key=%s", cfg.BaseURL, chID, url.QueryEscape(cfg.Key))
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("video-only TV was accepted: HTTP %d body=%q", resp.StatusCode, string(body))
	}
	if !bytes.Contains(body, []byte("mpegts-audio-track-not-found")) && !strings.Contains(string(body), "mpegts-audio-track-missing") {
		t.Fatalf("missing audio diagnostic: %q", body)
	}
}

// This fixture mirrors HLS broadcasters that carry video and AAC audio in
// separate playlists. Previous tests only exercised already-multiplexed TS
// segments, which cannot catch HLS audio rendition mapping regressions.
type splitAudioHLSTransport struct{ video, audio []byte }

func (t splitAudioHLSTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var content []byte
	mime := "application/vnd.apple.mpegurl"
	switch r.URL.Path {
	case "/live/master.m3u8":
		content = []byte(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",NAME="English",DEFAULT=YES,AUTOSELECT=YES,LANGUAGE="en",URI="audio.m3u8"
#EXT-X-STREAM-INF:BANDWIDTH=1800000,CODECS="avc1.42c01e,mp4a.40.2",AUDIO="audio"
video.m3u8
`)
	case "/live/video.m3u8":
		content = []byte("#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:3\n#EXT-X-MEDIA-SEQUENCE:0\n#EXTINF:3.0,\nvideo.ts\n#EXT-X-ENDLIST\n")
	case "/live/audio.m3u8":
		content = []byte("#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:3\n#EXT-X-MEDIA-SEQUENCE:0\n#EXTINF:3.0,\naudio.aac\n#EXT-X-ENDLIST\n")
	case "/live/video.ts":
		content = t.video
		mime = "video/mp2t"
	case "/live/audio.aac":
		content = t.audio
		mime = "audio/aac"
	default:
		return &http.Response{StatusCode: 404, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("missing")), Request: r}, nil
	}
	hdr := make(http.Header)
	hdr.Set("Content-Type", mime)
	return &http.Response{StatusCode: 200, Header: hdr, Body: io.NopCloser(bytes.NewReader(content)), ContentLength: int64(len(content)), Request: r}, nil
}

func TestMPEGTSSeparateAudioRendition(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not installed")
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe not installed")
	}
	video := makeTestTSPackets(t, false)
	audio, err := exec.Command(ffmpeg, "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000", "-t", "2", "-c:a", "aac", "-b:a", "64k", "-f", "adts", "pipe:1").Output()
	if err != nil {
		t.Skipf("cannot generate AAC: %v", err)
	}
	old := mediaClient
	mediaClient = &http.Client{Transport: splitAudioHLSTransport{video: video, audio: audio}, Timeout: 10 * time.Second}
	t.Cleanup(func() { mediaClient = old })
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cfg := exampleCfg()
	cfg.StreamMode = "mpegts"
	cfg.Bind = listener.Addr().String()
	cfg.BaseURL = "http://" + cfg.Bind
	ch := sampleFeed().Channels[0]
	ch.Stream = "https://streams.example.org/live/master.m3u8"
	server := &http.Server{Handler: newJellyfinHandler(cfg, func(string) ChannelData { return ChannelData{Channels: []Channel{ch}} }), ReadHeaderTimeout: 3 * time.Second}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	id := base64.RawURLEncoding.EncodeToString([]byte(ch.ID))
	endpoint := fmt.Sprintf("%s/jellyfin/stream/%s.ts?key=%s", cfg.BaseURL, id, url.QueryEscape(cfg.Key))
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	resp.Body.Close()
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("separate audio HLS: status=%d err=%v body=%q", resp.StatusCode, err, string(contents[:min(len(contents), 180)]))
	}
	if len(contents) < 188*20 {
		t.Fatalf("too little TS data %d", len(contents))
	}
	var probe tsAudioProbe
	for i := 0; i+188 <= len(contents); i += 188 {
		probe.add(contents[i : i+188])
	}
	if !probe.audioReady() {
		t.Fatalf("separate HLS audio not delivered: %+v", probe)
	}
	cmd := exec.Command(ffprobe, "-v", "error", "-f", "mpegts", "-show_entries", "stream=codec_name", "-of", "csv=p=0", "pipe:0")
	cmd.Stdin = bytes.NewReader(contents)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("ffprobe error: %v", err)
	}
	if !bytes.Contains(output, []byte("h264")) || !bytes.Contains(output, []byte("aac")) {
		t.Fatalf("separate audio mapping lost codec: %q", output)
	}
}
