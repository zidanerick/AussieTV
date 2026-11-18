//go:build !windows

package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fakeFFmpegScript(t *testing.T, script string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "ffmpeg-fake")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUSSIETV_FFMPEG_PATH", p)
	return p
}

func fakeMPEGTSRequest(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	cfg := exampleCfg()
	cfg.StreamMode = "mpegts"
	cfg.Bind = "127.0.0.1:45782"
	ch := sampleFeed().Channels[0]
	id := base64.RawURLEncoding.EncodeToString([]byte(ch.ID))
	u := fmt.Sprintf("/jellyfin/stream/%s.ts?key=%s", id, url.QueryEscape(cfg.Key))
	req := httptest.NewRequest("GET", u, nil)
	resp := httptest.NewRecorder()
	newJellyfinHandler(cfg, func(string) ChannelData { return ChannelData{Channels: []Channel{ch}} }).ServeHTTP(resp, req)
	return resp
}

func TestFailedFFmpegExitsWithSanitizedDiagnostic(t *testing.T) {
	fakeFFmpegScript(t, "echo 'https://secret.invalid/stream?token=never-leak: Server returned 403 Forbidden' >&2\nexit 1\n")
	before := len(tuneEvents())
	response := fakeMPEGTSRequest(t)
	if response.Code != 502 {
		t.Fatalf("expected 502, got %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "never-leak") {
		t.Fatal("secret in client response")
	}
	events := tuneEvents()
	if len(events) <= before || events[len(events)-1].Mode != "mpegts-source-forbidden" {
		t.Fatalf("missing safe failure event: %#v", events)
	}
	if activeMPEGTS.Load() != 0 {
		t.Fatal("orphan ffmpeg process")
	}
}

func TestHungFFmpegTimesOutAndIsStopped(t *testing.T) {
	fakeFFmpegScript(t, "while :; do :; done\n")
	old := mpegTSStartupTimeout
	mpegTSStartupTimeout = 180 * time.Millisecond
	t.Cleanup(func() { mpegTSStartupTimeout = old })
	started := time.Now()
	response := fakeMPEGTSRequest(t)
	if response.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d: %s", response.Code, response.Body.String())
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("startup timeout did not stop hung child: %v", elapsed)
	}
	if activeMPEGTS.Load() != 0 {
		t.Fatal("hung ffmpeg process still active")
	}
}
