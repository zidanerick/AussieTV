package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestDiscordActivityIsolation(t *testing.T) {
	calls := 0
	web := fstest.MapFS{"index.html": {Data: []byte("guide")}}
	h := discordActivityHandler(web, "123456789012345678", func(region string) ChannelData {
		calls++
		return ChannelData{Region: region, Channels: []Channel{
			{Name: "ABC TV", Network: "ABC", Stream: "secret-stream", Headers: map[string]string{"referer":"secret-header"}, Official:"https://example.invalid", Programs: []Program{{Start: 1, End: 2, Title: "News"}}},
			{Name:"Unknown", Network:"Other"},
		}}
	})
	for _, path := range []string{"/api/guide?region=Melbourne", "/.proxy/api/guide?region=Melbourne"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || strings.Contains(w.Body.String(),"secret") || strings.Contains(w.Body.String(),"example.invalid") {
			t.Fatalf("unsafe or failed guide response: %d %s", w.Code, w.Body)
		}
		var d discordGuide
		if err := json.Unmarshal(w.Body.Bytes(), &d); err != nil || len(d.Channels) != 1 || d.Channels[0].Official != official["ABC"] {
			t.Fatalf("invalid guide: %v %+v", err, d)
		}
	}
	if calls != 1 { t.Fatalf("expected one upstream load, got %d", calls) }
	for _, path := range []string{"/api/settings","/hls","/channel.m3u","/api/logo","/api/artwork","/teletext/100","/api/quit","/.proxy/hls","/config/teletext.json"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != http.StatusNotFound { t.Errorf("%s exposed: %d", path, w.Code) }
	}
	for _, method := range []string{"POST","PUT","DELETE"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(method, "/api/quit", nil))
		if w.Code < 400 { t.Errorf("%s quit exposed", method) }
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/guide?region=invalid", nil))
	if w.Code != 400 || calls != 1 { t.Fatal("invalid region reached upstream") }
}
