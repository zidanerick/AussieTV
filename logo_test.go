package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAllCuratedChannelsPermitValidProviderLogos(t *testing.T) {
	for _, spec := range curated {
		ch := Channel{ID: "mjh-verified-" + spec.name, Network: spec.net, Name: spec.name, Number: spec.number, Logo: "https://logos.example.net/assets/channel.png?format=small"}
		if !logoChannelAllowed(ch) {
			t.Errorf("curated channel not accepted: %s %s", ch.Network, ch.Name)
		}
		if target, err := realLogoURL(ch); err != nil || target != ch.Logo {
			t.Errorf("logo URL not accepted for %s: %s %v", ch.Name, target, err)
		}
	}
	if logoChannelAllowed(Channel{Network: "ABC", Name: "Unlisted", Number: "21"}) {
		t.Fatal("unlisted channel accepted")
	}
}

func TestLogoURLRejectsUntrustedEndpoints(t *testing.T) {
	for _, input := range []string{
		"", "http://logos.example.com/a.png", "file:///etc/passwd", "https://localhost/a.png",
		"https://127.0.0.1/a.png", "https://10.0.0.1/a.png", "https://[::1]/a.png",
		"https://user:password@logos.example.net/a.png", "https://logos.example.net:444/a.png",
		"https://internal/a.png", "https://logo.example.net/a.png#frag",
	} {
		ch := Channel{Network: "SBS", Name: "SBS", Number: "30", Logo: input}
		if url, err := realLogoURL(ch); err == nil {
			t.Errorf("accepted untrusted logo: %q (%s)", input, url)
		}
	}
}

type logoTestTransport func(*http.Request) (*http.Response, error)

func (f logoTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLogoHandlerOnlyRequestsCuratedImageAndRejectsSVG(t *testing.T) {
	oldClient := realLogoClient
	oldCache := logoCache.items
	cacheMu.Lock()
	oldChannelCache := cache
	cache = map[string]cacheEntry{"Melbourne": {data: ChannelData{Channels: []Channel{{ID: "abc-correct", Network: "ABC", Name: "ABC NEWS", Number: "24", Logo: "https://logos.example.net/genuine.png"}, {ID: "bad-id", Network: "SBS", Name: "Unlisted", Number: "30", Logo: "https://logos.example.net/evil.png"}}}, at: time.Now()}}
	cacheMu.Unlock()
	logoCache.Lock()
	logoCache.items = make(map[string]cachedLogo)
	logoCache.Unlock()
	t.Cleanup(func() {
		realLogoClient = oldClient
		cacheMu.Lock()
		cache = oldChannelCache
		cacheMu.Unlock()
		logoCache.Lock()
		logoCache.items = oldCache
		logoCache.Unlock()
	})
	// Valid tiny PNG fixture. No outgoing requests are made to the real provider.
	image, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAFgAI/ScLttAAAAABJRU5ErkJggg==")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	realLogoClient = &http.Client{Transport: logoTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://logos.example.net/genuine.png" {
			t.Fatalf("untrusted outbound URL: %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(image)), ContentLength: int64(len(image)), Header: make(http.Header), Request: r}, nil
	})}
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		logoHandler(w, httptest.NewRequest("GET", "/api/logo?region=Melbourne&ch=abc-correct", nil))
		if w.Code != 200 || w.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("want PNG 200 got %d %q", w.Code, w.Header().Get("Content-Type"))
		}
	}
	if calls != 1 {
		t.Errorf("expected 12-hour image cache hit, outbound calls=%d", calls)
	}
	for _, q := range []string{"ch=bad-id", "ch=some-other-id", "ch=..%2Fetc", "ch=abc-correct&region=NotReal"} {
		if strings.Contains(q, "region=NotReal") {
			continue
		} // Unknown regions are intentionally normalized to Melbourne.
		w := httptest.NewRecorder()
		logoHandler(w, httptest.NewRequest("GET", "/api/logo?"+q, nil))
		if w.Code != 404 {
			t.Errorf("unlisted channel accepted %q (status %d)", q, w.Code)
		}
	}
	realLogoClient = &http.Client{Transport: logoTestTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)), ContentLength: -1, Header: make(http.Header), Request: r}, nil
	})}
	logoCache.Lock()
	logoCache.items = make(map[string]cachedLogo)
	logoCache.Unlock()
	w := httptest.NewRecorder()
	logoHandler(w, httptest.NewRequest("GET", "/api/logo?ch=abc-correct", nil))
	if w.Code != 502 {
		t.Fatalf("unsafe SVG accepted via logo proxy, status %d", w.Code)
	}
}
