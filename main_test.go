package main

import (
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
