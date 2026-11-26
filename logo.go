package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const maxLogoBytes = 512 << 10

var realLogoClient = &http.Client{
	Timeout:       12 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	Transport: &http.Transport{
		Proxy:                 nil,
		DialContext:           safeDial,
		TLSHandshakeTimeout:   8 * time.Second,
		ResponseHeaderTimeout: 9 * time.Second,
	},
}

type cachedLogo struct {
	data   []byte
	mime   string
	stored time.Time
}

var logoCache = struct {
	sync.Mutex
	items map[string]cachedLogo
}{items: make(map[string]cachedLogo)}

// Only channels selected by the curated TV lineup may be proxied. The
// caller supplies a channel ID, never a URL, so this is not an open proxy.
func logoChannelAllowed(c Channel) bool {
	if ch,ok:=externalProviderChannel(c.ID);ok{return ch.Logo==c.Logo && ch.Name==c.Name && ch.Number==c.Number}

	for _, spec := range activeCurated() {
		if (c.Network == spec.net || (spec.net=="" && c.Network=="International")) && c.Name == spec.name && c.Number == spec.number {
			return true
		}
	}
	return false
}

// Use the guide's real logo URL for every channel, not hardcoded icon artwork.
// Source data may change, so accept only HTTPS with a standard port and no
// credentials. Public-IP DNS checks are enforced by safeDial; redirects are
// forbidden and callers cannot supply their own target URLs.
func realLogoURL(c Channel) (string, error) {
	if !logoChannelAllowed(c) {
		return "", errors.New("channel not in curated lineup")
	}
	raw := strings.TrimSpace(c.Logo)
	if raw == "" || len(raw) > 2048 {
		return "", errors.New("channel has no valid upstream logo")
	}
	uri, err := url.Parse(raw)
	if err != nil || uri == nil || uri.Scheme != "https" || uri.Hostname() == "" || uri.Port() != "" || uri.User != nil || uri.Fragment != "" || uri.Opaque != "" {
		return "", errors.New("invalid upstream image URL")
	}
	// The endpoint contains no user-supplied URL argument. Regardless,
	// reject IP literals and local domain names early, before DNS lookup.
	host := strings.TrimSuffix(strings.ToLower(uri.Hostname()), ".")
	if host == "localhost" || !strings.Contains(host, ".") || net.ParseIP(host) != nil {
		return "", errors.New("invalid upstream image host")
	}
	return uri.String(), nil
}

func renderLogo(w http.ResponseWriter, content []byte, mime string) {
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=900")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func serveRealLogo(w http.ResponseWriter, r *http.Request, channel Channel) {
	target, err := realLogoURL(channel)
	if err != nil {
		http.Error(w, "No supported channel logo", http.StatusNotFound)
		return
	}
	logoCache.Lock()
	cached, ok := logoCache.items[target]
	logoCache.Unlock()
	if ok && time.Since(cached.stored) < 12*time.Hour {
		renderLogo(w, cached.data, cached.mime)
		return
	}

	upstream, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		http.Error(w, "Invalid logo request", http.StatusBadGateway)
		return
	}
	upstream.Header.Set("Accept", "image/png, image/jpeg, image/webp, image/gif")
	resp, err := realLogoClient.Do(upstream)
	if err != nil {
		http.Error(w, "Logo host unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.ContentLength > maxLogoBytes {
		http.Error(w, "Logo image unavailable", http.StatusBadGateway)
		return
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxLogoBytes+1))
	if err != nil || len(data) > maxLogoBytes {
		http.Error(w, "Logo too large", http.StatusBadGateway)
		return
	}
	mime := http.DetectContentType(data)
	switch mime {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
	default:
		http.Error(w, fmt.Sprintf("Unsupported artwork content type: %s", mime), http.StatusBadGateway)
		return
	}
	logoCache.Lock()
	// Avoid unlimited growth if an upstream provider changes asset URLs.
	if len(logoCache.items) > 80 {
		logoCache.items = make(map[string]cachedLogo)
	}
	logoCache.items[target] = cachedLogo{data: data, mime: mime, stored: time.Now()}
	logoCache.Unlock()
	renderLogo(w, data, mime)
}

func logoHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("ch")
	if id == "" || len(id) > 100 || strings.ContainsAny(id, "/\\") {
		http.NotFound(w, r)
		return
	}
	region := safeRegion(r.URL.Query().Get("region"))
	for _, c := range loadChannels(region).Channels {
		if c.ID == id && logoChannelAllowed(c) {
			serveRealLogo(w, r, c)
			return
		}
	}
	http.NotFound(w, r)
}

// safeDial rejects private and loopback IPs; redirects are disabled.
