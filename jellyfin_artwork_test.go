package main

import (
	"encoding/base64"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testPosterURL = "https://static.tvmaze.com/uploads/images/medium_portrait/123/the-simpsons.jpg"

func TestXMLTVCachedArtworkAndNegativeCache(t *testing.T) {
	artMu.Lock()
	old := artCache
	artCache = map[string]artworkCacheEntry{
		canonicalTitle("The Simpsons"):   {Art: Artwork{Poster: testPosterURL}, At: time.Now()},
		canonicalTitle("Unknown Series"): {Art: Artwork{}, At: time.Now()},
		canonicalTitle("Stale Show"):     {Art: Artwork{Poster: testPosterURL}, At: time.Now().Add(-13 * time.Hour)},
	}
	artMu.Unlock()
	t.Cleanup(func() { artMu.Lock(); artCache = old; artMu.Unlock() })
	feed := sampleFeed()
	feed.Channels[0].Programs = []Program{
		{Start: 1731924000, End: 1731927600, Title: "The Simpsons"},
		{Start: 1731927600, End: 1731931200, Title: "Unknown Series"},
		{Start: 1731931200, End: 1731934800, Title: "Stale Show"},
		{Start: 1731934800, End: 1731938400, Title: "Morning News"},
	}
	output, err := xmltvFeed(feedChannels(feed), "Melbourne")
	if err != nil {
		t.Fatal(err)
	}
	var parsed xmltvDocument
	if err := xml.Unmarshal(output, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Channels) != 1 || parsed.Channels[0].Icon == nil || parsed.Channels[0].Icon.Src != feed.Channels[0].Logo {
		t.Fatalf("channel logo regression: %+v", parsed.Channels)
	}
	id := xmlTVID("Melbourne", feed.Channels[0])
	if parsed.Channels[0].ID != id || len(parsed.Programmes) != 4 {
		t.Fatalf("channel/programme mapping changed: %+v", parsed)
	}
	for i, item := range parsed.Programmes {
		if item.Channel != id {
			t.Fatalf("programme %d channel ID mismatch", i)
		}
		if i == 0 {
			if item.Icon == nil || item.Icon.Src != testPosterURL {
				t.Fatalf("cached artwork missing: %+v", item)
			}
			u, err := url.Parse(item.Icon.Src)
			if err != nil || u.Scheme != "https" || u.Hostname() != "static.tvmaze.com" {
				t.Fatalf("icon is not a valid trusted public URL: %s", item.Icon.Src)
			}
		} else if item.Icon != nil {
			t.Fatalf("uncached, negative or stale image exported: %+v", item)
		}
	}
}

func TestXMLTVLookupIsDeferredDeduplicatedAndAuthenticated(t *testing.T) {
	artMu.Lock()
	old := artCache
	artCache = map[string]artworkCacheEntry{}
	artMu.Unlock()
	t.Cleanup(func() { artMu.Lock(); artCache = old; artMu.Unlock() })
	oldClient := artworkClient
	var lookups atomic.Int32
	artworkClient = &http.Client{Transport: artworkTransport(func(req *http.Request) (*http.Response, error) {
		lookups.Add(1)
		if req.URL.Path != "/search/shows" || req.URL.Query().Get("q") != "The Simpsons" {
			t.Errorf("unexpected request %s", req.URL)
		}
		response := `[{"show":{"name":"The Simpsons","url":"https://www.tvmaze.com/shows/123/bluey","image":{"medium":"` + testPosterURL + `"}}}]`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(response)), Header: make(http.Header), Request: req}, nil
	})}
	t.Cleanup(func() { artworkClient = oldClient })
	feed := sampleFeed()
	feed.Channels[0].Programs = []Program{
		{Start: 1731924000, End: 1731927600, Title: "The Simpsons"},
		{Start: 1731927600, End: 1731931200, Title: "The Simpsons"},
		{Start: 1731931200, End: 1731934800, Title: "Morning News"},
	}
	cfg := exampleCfg()
	h := newJellyfinHandler(cfg, func(string) ChannelData { return feed })
	unauthorized := requestJellyfin(t, h, "/jellyfin/guide.xml", "GET")
	if unauthorized.Code != 401 {
		t.Fatalf("guide auth bypass: %d", unauthorized.Code)
	}
	start := time.Now()
	first := requestJellyfin(t, h, "/jellyfin/guide.xml?key="+url.QueryEscape(cfg.Key), "GET")
	if time.Since(start) > 200*time.Millisecond {
		t.Fatal("guide blocked on image lookup")
	}
	if first.Code != 200 {
		t.Fatalf("guide request failed %d", first.Code)
	}
	var parsed xmltvDocument
	if err := xml.Unmarshal(first.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	for _, programme := range parsed.Programmes {
		if programme.Icon != nil {
			t.Fatal("first uncached guide included icon")
		}
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if art, ok := cachedProgrammeArtwork("The Simpsons"); ok && art == testPosterURL {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if art, ok := cachedProgrammeArtwork("The Simpsons"); !ok || art != testPosterURL {
		t.Fatal("background artwork did not populate shared cache")
	}
	if lookups.Load() != 1 {
		t.Fatalf("duplicate artwork requests: %d", lookups.Load())
	}
	second := requestJellyfin(t, h, "/jellyfin/guide.xml?key="+url.QueryEscape(cfg.Key), "GET")
	parsed = xmltvDocument{}
	if err := xml.Unmarshal(second.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Programmes[0].Icon == nil || parsed.Programmes[0].Icon.Src != testPosterURL {
		t.Fatal("warm cache missing programme icon")
	}
	if lookups.Load() != 1 {
		t.Fatal("cached lookup repeated")
	}
	// Simulate the Jellyfin server fetching the exact emitted HTTPS asset.
	artURL := parsed.Programmes[0].Icon.Src
	u, _ := url.Parse(artURL)
	if !trustedPosterURL(artURL) || u.Host != "static.tvmaze.com" {
		t.Fatalf("invalid export URL: %s", artURL)
	}
}

func TestDefaultM3URedirectAndNoUnexpectedTS(t *testing.T) {
	cfg := exampleCfg()
	cfg.StreamMode = ""
	p := m3uFeed(feedChannels(sampleFeed()), cfg)
	if strings.Contains(p, ".ts?") || !strings.Contains(p, ".m3u8?key=") {
		t.Fatalf("default playlist changed: %s", p)
	}
	cfg.StreamMode = "mpegts"
	p = m3uFeed(feedChannels(sampleFeed()), cfg)
	if !strings.Contains(p, ".ts?key=") {
		t.Fatalf("opt-in TS playlist missing: %s", p)
	}
	id := base64.RawURLEncoding.EncodeToString([]byte(sampleFeed().Channels[0].ID))
	h := newJellyfinHandler(exampleCfg(), func(string) ChannelData { return sampleFeed() })
	r := requestJellyfin(t, h, "/jellyfin/stream/"+id+".ts?key="+url.QueryEscape(exampleCfg().Key), "GET")
	if r.Code != http.StatusTemporaryRedirect {
		t.Fatalf("default redirect changed %d", r.Code)
	}
}
