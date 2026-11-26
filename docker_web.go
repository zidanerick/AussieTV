package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type dockerWebConfig struct {
	bind string
	origin *url.URL
	username, password [32]byte
}

func dockerWebConfigFromEnv(serverMode bool) (*dockerWebConfig, error) {
	bind := strings.TrimSpace(os.Getenv("AUSSIETV_WEB_BIND"))
	if bind == "" { return nil, nil }
	if !serverMode { return nil, errors.New("AUSSIETV_WEB_BIND requires AUSSIETV_MODE=server") }
	if _, port, err := net.SplitHostPort(bind); err != nil || port == "" || port == "0" {
		return nil, errors.New("AUSSIETV_WEB_BIND must be a fixed host:port")
	}
	u, err := url.Parse(strings.TrimSpace(os.Getenv("AUSSIETV_WEB_ORIGIN")))
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return nil, errors.New("AUSSIETV_WEB_ORIGIN must be the exact http(s) browser origin, without a trailing slash")
	}
	user, pass := os.Getenv("AUSSIETV_WEB_USER"), os.Getenv("AUSSIETV_WEB_PASSWORD")
	if user == "" || strings.ContainsAny(user, ":\r\n") || len(pass) < 24 {
		return nil, errors.New("web UI requires AUSSIETV_WEB_USER and AUSSIETV_WEB_PASSWORD (at least 24 characters)")
	}
	return &dockerWebConfig{bind:bind, origin:u, username:sha256.Sum256([]byte(user)), password:sha256.Sum256([]byte(pass))}, nil
}

func (c *dockerWebConfig) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		// Never trust forwarded host or scheme headers.
		if !strings.EqualFold(r.Host, c.origin.Host) {
			http.Error(w, "Invalid web host", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != c.origin.String() {
			http.Error(w, "Invalid origin", http.StatusForbidden)
			return
		}
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			http.Error(w, "Cross-site request", http.StatusForbidden)
			return
		}
		user, pass, ok := r.BasicAuth()
		gotUser, gotPass := sha256.Sum256([]byte(user)), sha256.Sum256([]byte(pass))
		valid := subtle.ConstantTimeCompare(gotUser[:], c.username[:]) & subtle.ConstantTimeCompare(gotPass[:], c.password[:])
		if !ok || valid != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="AussieTV", charset="UTF-8"`)
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/heartbeat" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Read-only web UI", http.StatusMethodNotAllowed)
			return
		}
		// Explicit allow-list: no desktop management or playlist downloads.
		switch r.URL.Path {
		case "/":
			body, err := assets.ReadFile("static/index.html")
			if err != nil { http.Error(w, "Application asset missing", 500); return }
			body = bytes.Replace(body, []byte("</head>"), []byte("<style>a[href='/settings'],#quit,#playlist{display:none!important}</style></head>"), 1)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if r.Method == http.MethodGet { _, _ = w.Write(body) }
			return
		case "/api/channels", "/api/artwork", "/api/logo", "/hls", "/app-icon.svg", "/aussietv-logo.svg", "/remote.js", "/favicon.ico", "/favicon-32.png", "/favicon-192.png", "/site.webmanifest", "/channel-logo-abc-news.svg", "/channel-logo-sbs.svg", "/channel-logo-sbs2.svg", "/channel-logo-sbs-world-movies.svg", "/channel-logo-sbs-food.svg", "/channel-logo-nitv.svg":
			next.ServeHTTP(w,r)
		default:
			if strings.HasPrefix(r.URL.Path,"/teletext/") { next.ServeHTTP(w,r); return }
			http.NotFound(w,r)
		}
	})
}
