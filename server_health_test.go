package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServerHealthChecksBothListeners(t *testing.T) {
	t.Setenv("AUSSIETV_MODE", "server")
	t.Setenv("AUSSIETV_WEB_ORIGIN", "https://tv.example.test")
	t.Setenv("AUSSIETV_WEB_USER", "viewer")
	t.Setenv("AUSSIETV_WEB_PASSWORD", "test-password-123456789012345")
	t.Setenv("AUSSIETV_WEB_BIND", "127.0.0.1:45781")
	cfg, err := dockerWebConfigFromEnv(true)
	if err != nil { t.Fatal(err) }
	web := httptest.NewServer(cfg.handler(http.NotFoundHandler()))
	defer web.Close()
	t.Setenv("AUSSIETV_WEB_BIND", strings.TrimPrefix(web.URL,"http://"))
	const key = "test-feed-key-123456789012345"
	feed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
		if r.URL.Path != "/jellyfin/health" || r.URL.Query().Get("key") != key {
			http.Error(w,"unauthorized",401); return
		}
		w.WriteHeader(200)
	}))
	t.Setenv("AUSSIETV_JELLYFIN_BIND",strings.TrimPrefix(feed.URL,"http://"))
	t.Setenv("AUSSIETV_JELLYFIN_BASE_URL",feed.URL)
	t.Setenv("AUSSIETV_JELLYFIN_KEY",key)
	t.Setenv("AUSSIETV_JELLYFIN_STREAM_MODE","redirect")
	t.Setenv("AUSSIETV_HDHR_DISCOVERY","0")
	if err := serverHealthcheck(); err != nil { t.Fatal(err) }
	feed.Close()
	if err := serverHealthcheck(); err == nil || strings.Contains(err.Error(),key) { t.Fatalf("expected redacted feed failure: %v",err) }
	t.Setenv("AUSSIETV_JELLYFIN_BIND","")
	if err := serverHealthcheck(); err != nil { t.Fatal(err) }
	t.Setenv("AUSSIETV_WEB_PASSWORD","wrong-password-123456789012345")
	if err := serverHealthcheck(); err == nil { t.Fatal("wrong web password passed") }
}
