package main

import (
	"context"
	"net/http/httptest"
	"time"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCuratedChannelMapping(t *testing.T) {
	entries := map[string]Entry{
		"mjh-abc-vic":       {Name: "ABC TV", Network: "ABC", Master: "https://c.mjh.nz/abc-vic.m3u8", Programs: [][]any{{float64(2000), "Second show"}, {float64(1000), "First show"}}},
		"mjh-channel-9-vic": {Name: "Channel 9", Network: "Nine", Master: "https://i.mjh.nz/.r/channel-9-vic.m3u8"},
		"mjh-other":         {Name: "Sports Channel", Network: "Another Network", Master: "https://test.example/live.m3u8"},
	}
	d := makeData(entries, "Melbourne")
	if len(d.Channels) != 2 {
		t.Fatalf("got %d channels, want 2", len(d.Channels))
	}
	a := d.Channels[0]
	if a.Name != "ABC TV" || a.Number != "21" || a.ID != "mjh-abc-vic" {
		t.Fatalf("wrong channel %+v", a)
	}
	if len(a.Programs) != 2 || a.Programs[0].Title != "First show" || a.Programs[0].End != 2000 {
		t.Fatalf("programs not sorted or timed: %+v", a.Programs)
	}
	if d.Channels[1].Network != "Nine" {
		t.Fatalf("wrong ordering %+v", d.Channels)
	}
}
func TestRemoteUrlGuard(t *testing.T) {
	for _, url := range []string{"http://127.0.0.1/", "http://localhost/", "http://10.0.0.1/", "file:///etc/passwd", "http://[::1]/", "https://192.168.0.1/hls", "http://a.local/test"} {
		if _, err := validateRemote(url); err == nil {
			t.Errorf("failed to reject %s", url)
		}
	}
	for _, url := range []string{"https://c.mjh.nz/abc-vic.m3u8", "https://i.mjh.nz/.r/seven-mel.m3u8", "https://cdn.example.org/abc?token=1"} {
		if _, err := validateRemote(url); err != nil {
			t.Errorf("incorrectly rejected %s: %s", url, err)
		}
	}
}
func TestProviderFallback(t *testing.T) {
	d := fallback("Melbourne")
	if len(d.Channels) < 17 {
		t.Fatalf("missing default channels %d", len(d.Channels))
	}
	for _, c := range d.Channels {
		if !strings.HasPrefix(c.Official, "https://") {
			t.Fatalf("no official URL for %s", c.Name)
		}
	}
}

func TestArtworkTitleCleanup(t *testing.T) {
	tests := map[string]string{
		"NCIS S5 Ep. 11":         "NCIS",
		"Bluey S1 E03":           "Bluey",
		"The Chase Australia":    "The Chase Australia",
		"The Good Place (2016)":  "The Good Place",
		"7 News":                 "",
		"ABC Weather":            "",
		"":                       "",
		"NCIS: Hawai'i S3 Ep. 6": "NCIS: Hawai'i",
	}
	for in, want := range tests {
		if got := artworkTitle(in); got != want {
			t.Errorf("artworkTitle(%q) = %q; want %q", in, got, want)
		}
	}
}
func TestTrustedTVmazeArtwork(t *testing.T) {
	for _, raw := range []string{
		"http://static.tvmaze.com/img/poster.jpg", "https://example.com/poster.png",
		"https://static.tvmaze.com.evil.example/p.png", "file:///etc/passwd",
	} {
		if trustedPosterURL(raw) {
			t.Errorf("accepted unsafe poster URL %s", raw)
		}
	}
	if !trustedPosterURL("https://static.tvmaze.com/uploads/images/medium_portrait/123/abc.jpg") {
		t.Fatal("rejected official TVmaze image CDN")
	}
	if canonicalTitle("NCIS: Hawai'i") != canonicalTitle("NCIS Hawai i") {
		t.Fatal("title comparison should ignore punctuation")
	}
}

type artworkTransport func(*http.Request) (*http.Response, error)

func (f artworkTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
func TestArtworkExactMatching(t *testing.T) {
	old := artworkClient
	defer func() { artworkClient = old }()
	artworkClient = &http.Client{Transport: artworkTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "api.tvmaze.com" || req.URL.Query().Get("q") != "Bluey" {
			t.Errorf("unexpected lookup: %s", req.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"name":"Bluey","url":"https://www.tvmaze.com/shows/42692/bluey","image":{"medium":"https://static.tvmaze.com/uploads/images/medium_portrait/1/bluey.jpg"}}`)), Header: make(http.Header), Request: req}, nil
	})}
	result := lookupArtwork("Bluey")
	if result.Poster == "" || result.Credit != "TVmaze" || result.Page == "" {
		t.Fatalf("expected attributed image; got %+v", result)
	}
	artworkClient = &http.Client{Transport: artworkTransport(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"name":"Bluey: Extra Adventures","image":{"medium":"https://static.tvmaze.com/uploads/images/medium_portrait/1/bluey.jpg"}}`)), Header: make(http.Header), Request: req}, nil
	})}
	result = lookupArtwork("Bluey")
	if result.Poster != "" {
		t.Fatalf("accepted the wrong programme: %+v", result)
	}
}

func TestStreamProxyOnlyAllowsListedRootOrSignedManifestResources(t *testing.T) {
    channel := Channel{ID: "channel-abc", Stream: "https://cdn.example.net/live/master.m3u8"}
    if !permittedStream(channel, channel.Stream, "") {
        t.Fatal("listed channel stream must be accepted without a signature")
    }
    segment := "https://media.example.org/path/segment01.ts?auth=shortlived"
    if permittedStream(channel, segment, "") {
        t.Fatal("arbitrary untrusted segment was accepted")
    }
    signature := streamSignature(channel.ID, segment)
    if !permittedStream(channel, segment, signature) {
        t.Fatal("valid signed manifest resource was rejected")
    }
    if permittedStream(channel, segment+"&extra=1", signature) || permittedStream(Channel{ID: "channel-other"}, segment, signature) {
        t.Fatal("signature was reusable against a different URL or channel")
    }
    if permittedStream(channel, segment, "NOT-HEX") {
        t.Fatal("nonhex signature was accepted")
    }
    if link := playlistURL(channel.ID, segment); !strings.Contains(link, "&sig=") {
        t.Fatalf("playlist proxy URL is unsigned: %s", link)
    }
}

func TestHLSProxyRejectsUnlistedStreamBeforeFetching(t *testing.T) {
    cacheMu.Lock()
    old := cache
    cache = map[string]cacheEntry{"Test": {data: ChannelData{Channels: []Channel{{ID: "abc", Stream: "https://example.org/master.m3u8"}}}}}
    cacheMu.Unlock()
    defer func() { cacheMu.Lock(); cache = old; cacheMu.Unlock() }()
    request := httptest.NewRequest(http.MethodGet, "/hls?ch=abc&u=https%3A%2F%2Fexample.org%2Fforged.ts", nil)
    result := httptest.NewRecorder()
    hlsHandler(result, request)
    if result.Code != http.StatusForbidden {
        t.Fatalf("got HTTP %d, want 403 for forged address", result.Code)
    }
}

func TestSafeDialRejectsPrivateHostBeforeConnecting(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()
    conn, err := safeDial(ctx, "tcp", "localhost:80")
    if conn != nil { conn.Close(); t.Fatal("should not connect to localhost") }
    if err == nil { t.Fatal("did not reject localhost resolution") }
}

// Branding assets must be embedded in the executable and correctly typed.
func TestAppBrandingRoutes(t *testing.T) {
	cases := map[string]string{
		"/": "text/html", "/app-icon.svg": "image/svg+xml",
		"/aussietv-logo.svg": "image/svg+xml",
		"/favicon-32.png": "image/png", "/favicon-192.png": "image/png",
		"/favicon.ico": "image/x-icon",
		"/site.webmanifest": "application/manifest+json",
	}
	for path, typ := range cases {
		w := httptest.NewRecorder()
		serveAppAsset(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK { t.Errorf("GET %s status: %d", path, w.Code); continue }
		if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got,typ) { t.Errorf("GET %s content type: %s", path, got) }
		if w.Body.Len()==0 { t.Errorf("GET %s empty", path) }
	}
	w:=httptest.NewRecorder()
	serveAppAsset(w,httptest.NewRequest(http.MethodGet, "/not-a-route",nil))
	if w.Code!=http.StatusNotFound {t.Fatalf("unexpected route accepted: %d",w.Code)}
}
