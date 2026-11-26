package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDockerWebConfigAndIsolation(t *testing.T) {
	t.Setenv("AUSSIETV_WEB_BIND", "0.0.0.0:45781")
	t.Setenv("AUSSIETV_WEB_ORIGIN", "https://tv.example.test")
	t.Setenv("AUSSIETV_WEB_USER", "viewer")
	t.Setenv("AUSSIETV_WEB_PASSWORD", "a-long-test-password-123456")
	if _, err := dockerWebConfigFromEnv(false); err == nil { t.Fatal("desktop web bind accepted") }
	cfg, err := dockerWebConfigFromEnv(true)
	if err != nil { t.Fatal(err) }
	h := cfg.handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	tests := []struct{
		method,path,host,origin,site string
		auth bool
		code int
	}{
		{"GET","/api/channels","tv.example.test","","",false,401},
		{"GET","/api/health","tv.example.test","","",false,401},
		{"GET","/api/health","tv.example.test","","",true,200},
		{"GET","/api/channels","evil.test","","",true,403},
		{"GET","/api/channels","tv.example.test","https://evil.test","",true,403},
		{"GET","/hls","tv.example.test","","cross-site",true,403},
		{"GET","/hls","tv.example.test","https://tv.example.test","same-origin",true,200},
		{"GET","/api/settings","tv.example.test","","",true,404},
		{"GET","/settings","tv.example.test","","",true,404},
		{"GET","/channel.m3u","tv.example.test","","",true,404},
		{"GET","/unknown","tv.example.test","","",true,404},
		{"POST","/api/quit","tv.example.test","","",true,405},
		{"POST","/api/heartbeat","tv.example.test","https://tv.example.test","",true,204},
		{"GET","/","tv.example.test","","",true,200},
	}
	for _, tt := range tests {
		r := httptest.NewRequest(tt.method, "http://"+tt.host+tt.path, nil)
		if tt.auth { r.SetBasicAuth("viewer","a-long-test-password-123456") }
		r.Header.Set("Origin",tt.origin)
		r.Header.Set("Sec-Fetch-Site",tt.site)
		w := httptest.NewRecorder()
		h.ServeHTTP(w,r)
		if w.Code != tt.code { t.Errorf("%s %s: got %d want %d",tt.method,tt.path,w.Code,tt.code) }
		if tt.path=="/" && !strings.Contains(w.Body.String(),"#quit,#playlist") { t.Error("desktop controls not hidden") }
	}
	t.Setenv("AUSSIETV_WEB_PASSWORD","short")
	if _, err := dockerWebConfigFromEnv(true); err == nil { t.Fatal("weak password accepted") }
	t.Setenv("AUSSIETV_WEB_PASSWORD","a-long-test-password-123456")
	t.Setenv("AUSSIETV_WEB_ORIGIN","https://tv.example.test/path")
	if _, err := dockerWebConfigFromEnv(true); err == nil { t.Fatal("invalid origin accepted") }
	t.Setenv("AUSSIETV_WEB_BIND","")
	if c, err := dockerWebConfigFromEnv(true); c != nil || err != nil { t.Fatal("feed-only mode changed") }
}
