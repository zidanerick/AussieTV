package main

import (
	"aussietv/teletext"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestTeletextMainRouteAndJellyfinFeed(t *testing.T) {
	b, e := os.ReadFile("config/teletext.json")
	if e != nil {
		t.Fatal(e)
	}
	cfg, e := teletext.LoadConfig(b)
	if e != nil {
		t.Fatal(e)
	}
	saved := teletextService
	teletextService = teletext.New(cfg, nil)
	defer func() { teletextService = saved }()
	t.Setenv("AUSSIETV_TELETEXT_VIDEO", "1")
	if !teletext.HasVideoEncoder() {
		t.Skip("ffmpeg not installed")
	}
	fcfg := exampleCfg()
	handler := newJellyfinHandler(fcfg, func(string) ChannelData { return sampleFeed() })
	req := httptest.NewRequest("GET", "/jellyfin/channels.m3u?key="+fcfg.Key, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "aussietv.teletext") {
		t.Fatalf("text channel missing: %d %s", w.Code, w.Body.String())
	}
	req = httptest.NewRequest("GET", "/jellyfin/guide.xml?key="+fcfg.Key, nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "aussietv.teletext") {
		t.Fatalf("guide ID missing: %s", w.Body.String())
	}
	req = httptest.NewRequest("GET", "/teletext/page/100?key="+fcfg.Key, nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal("private text feed HTTP", w.Code)
	}
	var page teletext.Page
	if e = json.Unmarshal(w.Body.Bytes(), &page); e != nil || len(page.Rows) != 24 {
		t.Fatal(e)
	}
	req = httptest.NewRequest("GET", "/teletext/page/100", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatal("text feed leaked without key")
	}
	_ = io.Discard
}
