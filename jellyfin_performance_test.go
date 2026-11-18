package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

type blockedGuideTransport struct {
	started chan struct{}
	release chan struct{}
}

func (b *blockedGuideTransport) RoundTrip(*http.Request) (*http.Response, error) {
	select {
	case b.started <- struct{}{}:
	default:
	}
	<-b.release
	return nil, errors.New("simulated slow guide upstream")
}

func TestExpiredGuideDoesNotBlockJellyfinTune(t *testing.T) {
	// Reproduces the previous 20-24 second delay when the guide cache expired
	// during a Jellyfin HEAD/GET request. The old data is usable immediately.
	region := "Melbourne"
	cacheMu.Lock()
	previousCache, previousRunning, previousAttempt := cache, refreshRunning, refreshAttempt
	cache = map[string]cacheEntry{region: {data: sampleFeed(), at: time.Now().Add(-21 * time.Minute)}}
	refreshRunning = map[string]bool{}
	refreshAttempt = map[string]time.Time{}
	cacheMu.Unlock()
	previousClient := client
	blocker := &blockedGuideTransport{started: make(chan struct{}, 1), release: make(chan struct{})}
	client = &http.Client{Transport: blocker, Timeout: 5 * time.Second}
	t.Cleanup(func() {
		// Unblock and wait for refresh completion before restoring global state.
		close(blocker.release)
		deadline := time.After(2 * time.Second)
		for {
			cacheMu.Lock()
			busy := refreshRunning[region]
			cacheMu.Unlock()
			if !busy {
				break
			}
			select {
			case <-deadline:
				t.Error("background refresh did not finish")
				return
			default:
				time.Sleep(time.Millisecond)
			}
		}
		client = previousClient
		cacheMu.Lock()
		cache, refreshRunning, refreshAttempt = previousCache, previousRunning, previousAttempt
		cacheMu.Unlock()
	})
	start := time.Now()
	data := loadChannels(region)
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("stale guide blocked channel tuning for %s", elapsed)
	}
	if len(data.Channels) == 0 || data.Channels[0].ID != sampleFeed().Channels[0].ID {
		t.Fatal("stale guide not returned")
	}
	select {
	case <-blocker.started:
	case <-time.After(time.Second):
		t.Fatal("async background refresh did not begin")
	}
	start = time.Now()
	data = loadChannels(region)
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("second request blocked: %s", elapsed)
	}
	if len(data.Channels) == 0 {
		t.Fatal("missing cached channels")
	}
}

func TestHDHRPort80FailureKeepsManualTunerAvailable(t *testing.T) {
	cfg := exampleCfg()
	cfg.Bind = "127.0.0.1:0"               // ephemeral authenticated M3U feed
	cfg.BaseURL = "http://127.0.0.1:45782" // not eligible for LAN port 80 discovery
	cfg.Discovery = true
	supervisor := &feedSupervisor{}
	defer supervisor.Close()
	if err := supervisor.Replace(cfg, true); err != nil {
		t.Fatalf("discovery error must not stop the M3U feed: %v", err)
	}
	if !strings.Contains(supervisor.Status(), "Auto-detection unavailable") {
		t.Fatalf("missing warning: %s", supervisor.Status())
	}
	if supervisor.discoveryConn != nil || supervisor.hdhrListener != nil {
		t.Fatal("incorrectly enabled discovery on loopback")
	}
}
