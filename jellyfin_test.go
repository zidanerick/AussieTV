package main

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func sampleFeed() ChannelData {
	return ChannelData{Region: "Melbourne", Channels: []Channel{
		{ID: "mjh:abc/news", Name: "ABC News & Current Affairs", Number: "24", Network: "ABC", Logo: "https://images.example.org/logo.png", Stream: "https://streams.example.org/live.m3u8?token=demo", Programs: []Program{{Start: 1731924000, End: 1731927600, Title: "News & Analysis <Special>"}, {Start: 1731927600, End: 1731931200, Title: "Evening News"}}},
		{ID: "no-stream", Name: "Unavailable", Number: "11", Network: "Ten"},
		{ID: "private", Name: "Bad provider", Number: "99", Network: "ABC", Stream: "http://127.0.0.1/internal.m3u8"},
	}}
}
func exampleCfg() jellyfinConfig {
	return jellyfinConfig{Bind: "127.0.0.1:45782", BaseURL: "http://192.168.1.50:45782", Key: "a-strong-testing-key-over-24-characters", Region: "Melbourne"}
}
func requestJellyfin(t *testing.T, h http.Handler, path string, method string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}
func TestJellyfinM3UXMLTVMatch(t *testing.T) {
	cfg := exampleCfg()
	h := newJellyfinHandler(cfg, func(s string) ChannelData {
		if s != "Melbourne" {
			t.Fatalf("wrong region: %s", s)
		}
		return sampleFeed()
	})
	playlist := requestJellyfin(t, h, "/jellyfin/channels.m3u?key="+url.QueryEscape(cfg.Key), "GET")
	if playlist.Code != 200 || !strings.Contains(playlist.Body.String(), "#EXTM3U") {
		t.Fatalf("M3U response: %d %s", playlist.Code, playlist.Body.String())
	}
	id := xmlTVID(cfg.Region, sampleFeed().Channels[0])
	if !strings.Contains(playlist.Body.String(), `tvg-id="`+id+`"`) {
		t.Fatalf("missing ID: %s", playlist.Body.String())
	}
	if strings.Contains(playlist.Body.String(), "Unavailable") || strings.Contains(playlist.Body.String(), "Bad provider") {
		t.Fatalf("bad channel listed")
	}
	xmlr := requestJellyfin(t, h, "/jellyfin/guide.xml?key="+url.QueryEscape(cfg.Key), "GET")
	var parsed xmltvDocument
	if err := xml.Unmarshal(xmlr.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Channels) != 1 || len(parsed.Programmes) != 2 || parsed.Channels[0].ID != id || parsed.Programmes[0].Channel != id {
		t.Fatalf("guide mismatch: %+v", parsed)
	}
	if parsed.Programmes[0].Title.Value != "News & Analysis <Special>" {
		t.Fatalf("XML escaping broken")
	}
	if !strings.Contains(xmlr.Body.String(), "&amp;") || !strings.Contains(xmlr.Body.String(), "&lt;Special&gt;") {
		t.Fatalf("unsafe guide XML")
	}
	if parsed.Programmes[0].Start != xmltvTime(1731924000) {
		t.Fatalf("wrong timezone")
	}
	encoded := base64.RawURLEncoding.EncodeToString([]byte("mjh:abc/news"))
	redir := requestJellyfin(t, h, "/jellyfin/stream/"+encoded+"?key="+url.QueryEscape(cfg.Key), "GET")
	if redir.Code != http.StatusTemporaryRedirect || redir.Header().Get("Location") != "https://streams.example.org/live.m3u8?token=demo" {
		t.Fatalf("bad stream redirect: %d %v", redir.Code, redir.Header())
	}
}
func TestJellyfinAuthAndIsolation(t *testing.T) {
	cfg := exampleCfg()
	h := newJellyfinHandler(cfg, func(string) ChannelData { return sampleFeed() })
	for _, path := range []string{"/jellyfin/channels.m3u", "/jellyfin/guide.xml", "/jellyfin/stream/xyz", "/jellyfin/health"} {
		if result := requestJellyfin(t, h, path, "GET"); result.Code != 401 {
			t.Fatalf("%s unauthenticated: %d", path, result.Code)
		}
	}
	for _, path := range []string{"/api/quit", "/api/channels", "/hls?u=foo", "/"} {
		if result := requestJellyfin(t, h, path+map[bool]string{true: "&key=", false: "?key="}[strings.Contains(path, "?")]+url.QueryEscape(cfg.Key), "GET"); result.Code != 404 {
			t.Fatalf("%s leaked desktop endpoint: %d", path, result.Code)
		}
	}
	bad := requestJellyfin(t, h, "/jellyfin/stream/!!!?key="+url.QueryEscape(cfg.Key), "GET")
	if bad.Code != 404 {
		t.Fatal("invalid stream identifier should 404")
	}
	bad = requestJellyfin(t, h, "/jellyfin/stream/"+base64.RawURLEncoding.EncodeToString([]byte("private"))+"?key="+url.QueryEscape(cfg.Key), "GET")
	if bad.Code != 404 {
		t.Fatal("private URL must be disallowed")
	}
	forbidden := requestJellyfin(t, h, "/jellyfin/channels.m3u?key="+url.QueryEscape(cfg.Key), "POST")
	if forbidden.Code != 405 {
		t.Fatalf("unexpected method response %d", forbidden.Code)
	}
}
func TestJellyfinXMLTVTimeAndSanitizing(t *testing.T) {
	ch := []Channel{{ID: "1\"\n", Name: "Injected\n#EXTINF:-1", Number: "1\"", Network: "ABC", Stream: "https://example.org/live.m3u8"}}
	out := m3uFeed(ch, exampleCfg())
	if bytes.Count([]byte(out), []byte("\n#EXTINF:-1")) != 1 {
		t.Fatalf("playlist metadata injection: %s", out)
	}
	if strings.Contains(out, `tvg-name="Injected`+"\n") {
		t.Fatal("newline injection")
	}
	when := xmltvTime(time.Date(2026, 11, 18, 4, 0, 0, 0, time.UTC).Unix())
	if when != "20261118040000 -0000" {
		t.Fatalf("timestamp: %s", when)
	}
}
