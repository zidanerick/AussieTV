package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesktopSettingsValidation(t *testing.T) {
	cfg := defaultSettings()
	if err := validateSettings(cfg); err != nil {
		t.Fatalf("default invalid: %v", err)
	}
	cfg.Enabled = true
	cfg.LAN = true
	cfg.Discovery = true
	cfg.BaseURL = "http://192.168.2.12:45782"
	cfg.Key = "some-long-settings-credential-over-24-chars"
	if err := validateSettings(cfg); err != nil {
		t.Fatal(err)
	}
	cfg.LAN = false
	if err := validateSettings(cfg); err == nil {
		t.Fatal("discovery without LAN accepted")
	}
	cfg.LAN = true
	cfg.BaseURL = "http://127.0.0.1:45782"
	if err := validateSettings(cfg); err == nil {
		t.Fatal("LAN with loopback advertised URL accepted")
	}
}
func TestSettingsAPIRequiresCSRFAndPersists(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	supervisor := &feedSupervisor{}
	ctrl, err := newSettingsController(supervisor, false)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	ctrl.handler(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	var old publicSettings
	if err := json.Unmarshal(w.Body.Bytes(), &old); err != nil {
		t.Fatal(err)
	}
	if old.Enabled || old.Discovery {
		t.Fatal("fresh settings should be disabled")
	}
	post := func(token string) *httptest.ResponseRecorder {
		body := []byte(`{"enabled":false,"lan":false,"discovery":false,"baseUrl":"http://127.0.0.1:45782","region":"Sydney"}`)
		r := httptest.NewRequest(http.MethodPost, "/api/settings", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-AussieTV-CSRF", token)
		w := httptest.NewRecorder()
		ctrl.handler(w, r)
		return w
	}
	if post("").Code != 403 {
		t.Fatal("CSRF missing accepted")
	}
	if w := post(old.CSRF); w.Code != 200 {
		t.Fatalf("save failed: %d %s", w.Code, w.Body.String())
	}
	p := filepath.Join(os.Getenv("LOCALAPPDATA"), "AussieTV", "settings.json")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"region": "Sydney"`) {
		t.Fatalf("not saved: %s", data)
	}
}
func TestHDHRAuthorizationMode(t *testing.T) {
	cfg := exampleCfg()
	for _, discovery := range []bool{false, true} {
		cfg.Discovery = discovery
		h := newJellyfinHandler(cfg, func(string) ChannelData { return sampleFeed() })
		req := httptest.NewRequest("GET", "/discover.json", nil)
		req.RemoteAddr = "192.168.1.3:50711"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		want := 401
		if discovery {
			want = 200
		}
		if w.Code != want {
			t.Fatalf("enabled=%t got=%d want=%d", discovery, w.Code, want)
		}
		req = httptest.NewRequest("GET", "/jellyfin/channels.m3u", nil)
		req.RemoteAddr = "192.168.1.3:50711"
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != 401 {
			t.Fatalf("keyed M3U leaked in discovery mode: %d", w.Code)
		}
	}
}
