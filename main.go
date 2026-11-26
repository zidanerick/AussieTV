package main

import (
	"aussietv/teletext"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

//go:embed static/* config/teletext.json
var assets embed.FS

const sourceBase = "https://i.mjh.nz/au/"
const version = "0.1.0-beta.1"

type Entry struct {
	Chno     any               `json:"chno,omitempty"`
	Name     string            `json:"name"`
	Network  string            `json:"network"`
	Logo     string            `json:"logo"`
	Master   string            `json:"mjh_master"`
	Headers  map[string]string `json:"headers"`
	Programs [][]any           `json:"programs"`
}
type Program struct {
	Start int64  `json:"start"`
	End   int64  `json:"end"`
	Title string `json:"title"`
}
type Channel struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Number   string            `json:"number"`
	Network  string            `json:"network"`
	Logo     string            `json:"logo,omitempty"`
	Stream   string            `json:"stream,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Official string            `json:"official"`
	Programs []Program         `json:"programs"`
}
type ChannelData struct {
	Region   string    `json:"region"`
	Updated  string    `json:"updated"`
	Source   string    `json:"source"`
	Warning  string    `json:"warning,omitempty"`
	Channels []Channel `json:"channels"`
}

var curated = []struct{ net, name, number string }{
	{"ABC", "ABC TV", "21"}, {"ABC", "ABC NEWS", "24"}, {"ABC", "ABC Kids", "22"}, {"ABC", "ABC Entertains", "23"}, {"ABC", "ABC Family", "22"},
	{"SBS", "SBS", "30"}, {"SBS", "SBS2", "31"}, {"SBS", "SBS World Movies", "32"}, {"SBS", "SBS Food", "33"}, {"SBS", "NITV", "34"},
	{"Seven", "Seven", "7"}, {"Seven", "7two", "72"}, {"Seven", "7mate", "73"}, {"Seven", "7flix", "76"},
	{"Nine", "Channel 9", "9"}, {"Nine", "9Gem", "92"}, {"Nine", "9Go!", "93"}, {"Nine", "9Life", "94"},
	{"Ten", "10", "10"}, {"Ten", "10 Drama", "12"}, {"Ten", "10 Comedy", "11"}, {"Ten", "Nickelodeon", "13"},
}
var regionAllowed = map[string]bool{"Melbourne": true, "Sydney": true, "Brisbane": true, "Adelaide": true, "Perth": true, "Hobart": true, "Canberra": true, "Darwin": true}
var official = map[string]string{
	"ABC":   "https://iview.abc.net.au/collection/959",
	"SBS":   "https://www.sbs.com.au/ondemand/live",
	"Seven": "https://7plus.com.au/live-tv",
	"Nine":  "https://www.9now.com.au/live/channel-9",
	"Ten":   "https://10.com.au/live",
}
var client = &http.Client{Timeout: 24 * time.Second, Transport: &http.Transport{
	Proxy: http.ProxyFromEnvironment, MaxIdleConns: 50, MaxIdleConnsPerHost: 12, IdleConnTimeout: 65 * time.Second,
	TLSHandshakeTimeout: 12 * time.Second, ResponseHeaderTimeout: 20 * time.Second,
}}

// Each install's local HLS proxy only accepts the indexed channel's root URL,
// or URLs that the backend itself rewrote from an upstream HLS manifest.
var streamKey [32]byte

func init() {
	if _, err := rand.Read(streamKey[:]); err != nil {
		panic("cannot initialize the stream proxy secret: " + err.Error())
	}
}

func safeDial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(lookupCtx, host)
	if err != nil {
		return nil, err
	}
	// Do not permit DNS names to resolve to LAN, loopback or metadata targets.
	var dialer net.Dialer
	for _, target := range ips {
		ip := target.IP
		if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		err = dialErr
	}
	if err == nil {
		err = errors.New("stream target resolves only to blocked IP addresses")
	}
	return nil, err
}

var mediaClient = &http.Client{Timeout: 0, CheckRedirect: func(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("too many stream redirects")
	}
	_, err := validateRemote(req.URL.String())
	return err
}, Transport: &http.Transport{
	// Avoid a system proxy bypassing the remote IP restrictions.
	DialContext: safeDial, MaxIdleConns: 80, MaxIdleConnsPerHost: 24,
	IdleConnTimeout: 45 * time.Second, TLSHandshakeTimeout: 12 * time.Second, ResponseHeaderTimeout: 25 * time.Second,
}}

func streamSignature(channelID, remote string) string {
	mac := hmac.New(sha256.New, streamKey[:])
	_, _ = io.WriteString(mac, channelID+"\n"+remote)
	return hex.EncodeToString(mac.Sum(nil))
}
func permittedStream(ch Channel, remote, signature string) bool {
	if ch.Stream != "" && remote == ch.Stream {
		return true
	}
	if signature == "" {
		return false
	}
	provided, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, streamKey[:])
	_, _ = io.WriteString(mac, ch.ID+"\n"+remote)
	return subtle.ConstantTimeCompare(mac.Sum(nil), provided) == 1
}

type cacheEntry struct {
	data ChannelData
	at   time.Time
}

var cacheMu sync.Mutex
var cache = map[string]cacheEntry{}
var refreshRunning = map[string]bool{}
var refreshAttempt = map[string]time.Time{}

var lastPing atomic.Int64
var hadPing atomic.Bool
var trayReady atomic.Bool
var server *http.Server
var uriRegexp = regexp.MustCompile(`URI="([^"]+)"`)

// The guide feed contains channel logos, but not programme images. For clearly
// identified TV series, look up the poster at TVmaze and show its attribution.
// An approximate/ambiguous result is intentionally treated as no artwork.
type Artwork struct {
	Poster string `json:"poster,omitempty"`
	Backdrop string `json:"backdrop,omitempty"`
	Credit string `json:"credit,omitempty"`
	Page   string `json:"page,omitempty"`
}
type artworkCacheEntry struct {
	Art Artwork
	At  time.Time
}

var artMu sync.Mutex
var artCache = make(map[string]artworkCacheEntry)
var seasonSuffix = regexp.MustCompile(`(?i)\s+(?:s\d{1,3}\s*(?:ep\.?|e)\s*\d+.*|season\s+\d+.*|series\s+\d+.*)$`)
var titlePunctuation = regexp.MustCompile(`[^\pL\pN]+`)
var titleYear = regexp.MustCompile(`\s+\(\d{4}\)$`)
var artworkClient = &http.Client{Timeout: 9 * time.Second}

func artworkTitle(raw string) string {
	title := strings.TrimSpace(raw)
	title = seasonSuffix.ReplaceAllString(title, "")
	title = titleYear.ReplaceAllString(title, "")
	title = strings.TrimSpace(title)
	// Live bulletin names frequently collide with scripted series names.
	low := strings.ToLower(title)
	if len([]rune(title)) < 4 || len([]rune(title)) > 95 ||
		strings.Contains(low, "news") || strings.Contains(low, "weather") ||
		strings.Contains(low, "live stream") || strings.Contains(low, "bulletin") ||
		strings.Contains(low, "sport update") || strings.HasPrefix(low, "national lottery") {
		return ""
	}
	return title
}
func canonicalTitle(s string) string {
	return strings.TrimSpace(titlePunctuation.ReplaceAllString(strings.ToLower(s), " "))
}
func trustedPosterURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Hostname() == "static.tvmaze.com" && u.User == nil
}
func trustedTVmazeURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Hostname() == "www.tvmaze.com" && u.User == nil
}
func lookupArtwork(title string) Artwork {
	// A single-search result can silently select one of several identically
	// named shows. Search all candidates and only accept one exact match.
	var empty Artwork
	req, err := http.NewRequest("GET", "https://api.tvmaze.com/search/shows?q="+url.QueryEscape(title), nil)
	if err != nil {
		return empty
	}
	req.Header.Set("User-Agent", "AussieTV/0.1.0-beta.1 (TVmaze artwork attribution)")
	response, err := pacedArtworkRequest(req)
	if err != nil {
		log.Printf("programme art lookup: %v", err)
		return empty
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return empty
	}
	var candidates []struct {
		Show struct {
			ID int `json:"id"`
			Name  string `json:"name"`
			URL   string `json:"url"`
			Image struct {
				Medium   string `json:"medium"`
				Original string `json:"original"`
			} `json:"image"`
		} `json:"show"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 512<<10)).Decode(&candidates); err != nil {
		return empty
	}
	var match *Artwork
	var showID int
	for _, candidate := range candidates {
		show := candidate.Show
		if canonicalTitle(show.Name) != canonicalTitle(title) {
			continue
		}
		// Even an exact name is ambiguous when the catalogue lists two shows.
		if match != nil {
			return empty
		}
		poster := show.Image.Medium
		if !trustedPosterURL(poster) && trustedPosterURL(show.Image.Original) {
			poster = show.Image.Original
		}
		art := Artwork{}
		if trustedPosterURL(poster) {
			art = Artwork{Poster: poster, Credit: "TVmaze"}
			if trustedTVmazeURL(show.URL) {
				art.Page = show.URL
			}
		}
		match = &art
		showID = show.ID
	}
	if match == nil {
		return empty
	}
	if showID > 0 && match.Poster != "" {
		match.Backdrop = lookupArtworkBackdrop(showID)
	}
	return *match
}

// These workers share the same cache as /api/artwork. An XMLTV HTTP request
// never waits for an image API call. Duplicate/missing titles have a bounded
// queue; lookups are paced below TVmaze's published unauthenticated limit.
var artworkBackground = struct {
	sync.Mutex
	once    sync.Once
	pending map[string]bool
	queue   chan string
}{pending: make(map[string]bool), queue: make(chan string, 128)}

func artworkCacheIsFresh(entry artworkCacheEntry) bool {
	ttl := 12 * time.Hour
	// Negative and transient API failures should be retried sooner.
	if entry.Art.Poster == "" {
		ttl = 30 * time.Minute
	}
	return time.Since(entry.At) < ttl
}

func cachedProgrammeArtwork(title string) (string, bool) {
	clean := artworkTitle(title)
	if clean == "" {
		return "", false
	}
	key := canonicalTitle(clean)
	artMu.Lock()
	entry, found := artCache[key]
	artMu.Unlock()
	if !found || !artworkCacheIsFresh(entry) {
		return "", false
	}
	if !trustedPosterURL(entry.Art.Poster) {
		return "", true
	} // cached miss
	return entry.Art.Poster, true
}

func scheduleProgrammeArtwork(channels []Channel) {
	seen := make(map[string]bool)
	for _, programme := range prioritizedArtworkProgrammes(channels) {
			clean := artworkTitle(programme.Title)
			if clean == "" {
				continue
			}
			key := canonicalTitle(clean)
			if seen[key] {
				continue
			}
			seen[key] = true
			if _, cached := cachedProgrammeArtwork(clean); cached {
				continue
			}
			artworkBackground.Lock()
			if artworkBackground.pending[key] {
				artworkBackground.Unlock()
				continue
			}
			artworkBackground.pending[key] = true
			select {
			case artworkBackground.queue <- clean:
			default:
				delete(artworkBackground.pending, key)
			}
			artworkBackground.Unlock()
	}
	artworkBackground.once.Do(func() {
				for i := 0; i < 2; i++ {
			go func() {
				for title := range artworkBackground.queue {
					key := canonicalTitle(title)
					// A concurrent browser lookup might have populated it.
					if _, exists := cachedProgrammeArtwork(title); !exists {
						result := lookupArtwork(title)
						storeArtworkCacheEntry(key, result)
					}
					artworkBackground.Lock()
					delete(artworkBackground.pending, key)
					artworkBackground.Unlock()
				}
			}()
		}
	})
}

func artworkQueueStats() (posters int, misses int, pending int) {
	artMu.Lock()
	for _, entry := range artCache {
		if !artworkCacheIsFresh(entry) {
			continue
		}
		if trustedPosterURL(entry.Art.Poster) {
			posters++
		} else {
			misses++
		}
	}
	artMu.Unlock()
	artworkBackground.Lock()
	pending = len(artworkBackground.pending)
	artworkBackground.Unlock()
	return
}

func artworkHandler(w http.ResponseWriter, r *http.Request) {
	title := artworkTitle(r.URL.Query().Get("title"))
	if title == "" {
		jsonResponse(w, Artwork{})
		return
	}
	key := canonicalTitle(title)
	artMu.Lock()
	cached, exists := artCache[key]
	artMu.Unlock()
	if exists && artworkCacheIsFresh(cached) {
		jsonResponse(w, cached.Art)
		return
	}
	result := lookupArtwork(title)
	storeArtworkCacheEntry(key, result)
	jsonResponse(w, result)
}

func safeRegion(v string) string {
	if regionAllowed[v] {
		return v
	}
	return "Melbourne"
}
func loadChannels(region string) ChannelData {
	region = safeRegion(region)
	cacheMu.Lock()
	old, ok := cache[region]
	if ok {
		if time.Since(old.at) >= 20*time.Minute && !refreshRunning[region] && time.Since(refreshAttempt[region]) > time.Minute {
			refreshRunning[region] = true
			refreshAttempt[region] = time.Now()
			go func() {
				defer func() {
					cacheMu.Lock()
					refreshRunning[region] = false
					cacheMu.Unlock()
				}()
				// Work outside cacheMu. A failing upstream must not stall tuning.
				_ = fetchChannels(region, old, true)
			}()
		}
		cacheMu.Unlock()
		return old.data
	}
	cacheMu.Unlock()
	return fetchChannels(region, cacheEntry{}, false)
}

func fetchChannels(region string, old cacheEntry, hasOld bool) ChannelData {
	req, _ := http.NewRequest("GET", sourceBase+region+"/tv.json", nil)
	req.Header.Set("User-Agent", "AussieTV/"+version+" (+personal local player)")
	response, err := client.Do(req)
	if err == nil && response.StatusCode != 200 {
		err = fmt.Errorf("EPG server HTTP %d", response.StatusCode)
		response.Body.Close()
	}
	if err == nil {
		defer response.Body.Close()
		var entries map[string]Entry
		err = json.NewDecoder(io.LimitReader(response.Body, 6<<20)).Decode(&entries)
		if err == nil {
			data := makeData(entries, region)
			if len(data.Channels) > 0 {
				cacheMu.Lock()
				cache[region] = cacheEntry{data: data, at: time.Now()}
				cacheMu.Unlock()
				saveOffline(region, data)
				return data
			}
			err = errors.New("no recognised channels returned")
		}
	}
	if hasOld {
		old.data.Warning = "Guide refresh unavailable; using cached listings. Streaming may still work."
		return old.data
	}
	if offline, loaderr := loadOffline(region); loaderr == nil {
		offline.Warning = "Guide offline; displaying last saved channel data."
		return offline
	}
	data := fallback(region)
	data.Warning = "Could not retrieve live guide; channel lineup and stream links are unverified fallbacks."
	if err != nil {
		log.Printf("EPG refresh error: %v", err)
	}
	return data
}

// Only preserve safe provider HTTP headers for indexed channels. Never accept
// arbitrary upstream-supplied Host, Authorization, cookies, or proxy headers.
func providerHeaders(headers map[string]string) map[string]string {
	out := map[string]string{}
	ua := strings.TrimSpace(headers["user-agent"])
	if ua != "" && len(ua) <= 256 && printableHeaderValue(ua) {
		out["user-agent"] = ua
	}
	referer := strings.TrimSpace(headers["referer"])
	if referer != "" && len(referer) <= 1024 && printableHeaderValue(referer) {
		if _, err := validateRemote(referer); err == nil {
			out["referer"] = referer
		}
	}
	return out
}
func printableHeaderValue(s string) bool {
	for _, c := range s {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}
func setProviderRequestHeaders(req *http.Request, ch Channel) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/124.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	for k, v := range providerHeaders(ch.Headers) {
		switch k {
		case "user-agent":
			req.Header.Set("User-Agent", v)
		case "referer":
			req.Header.Set("Referer", v)
		}
	}
}

func makeData(entries map[string]Entry, region string) ChannelData {
	result := ChannelData{Region: region, Updated: time.Now().Format(time.RFC3339), Source: sourceBase + region + "/tv.json", Channels: []Channel{}}
	for _, spec := range curated {
		var id string
		var entry Entry
		for k, e := range entries {
			if e.Network == spec.net && e.Name == spec.name {
				id = k
				entry = e
				break
			}
		}
		if id == "" {
			continue
		}
		ch := Channel{ID: id, Name: entry.Name, Number: spec.number, Network: entry.Network, Logo: entry.Logo, Stream: entry.Master, Headers: providerHeaders(entry.Headers), Official: official[spec.net], Programs: []Program{}}
		var points []Program
		for _, p := range entry.Programs {
			if len(p) < 2 {
				continue
			}
			start, ok := p[0].(float64)
			title, ok2 := p[1].(string)
			if ok && ok2 {
				points = append(points, Program{Start: int64(start), Title: title})
			}
		}
		sort.Slice(points, func(i, j int) bool { return points[i].Start < points[j].Start })
		for i := range points {
			if i+1 < len(points) {
				points[i].End = points[i+1].Start
			} else {
				points[i].End = points[i].Start + 3600
			}
		}
		ch.Programs = points
		result.Channels = append(result.Channels, ch)
	}
	return result
}
func fallback(region string) ChannelData {
	data := ChannelData{Region: region, Updated: "", Source: "Offline fallback", Channels: []Channel{}}
	for _, c := range curated {
		data.Channels = append(data.Channels, Channel{ID: "fallback-" + strings.ReplaceAll(strings.ToLower(c.net+"-"+c.name), " ", "-"), Name: c.name, Number: c.number, Network: c.net, Official: official[c.net], Programs: []Program{}})
	}
	return data
}
func cachePath(region string) string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "AussieTV", region+".json")
}
func saveOffline(region string, data ChannelData) {
	path := cachePath(region)
	if path == "" {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0700) != nil {
		return
	}
	b, err := json.Marshal(data)
	if err == nil {
		_ = os.WriteFile(path, b, 0600)
	}
}
func loadOffline(region string) (ChannelData, error) {
	var d ChannelData
	b, err := os.ReadFile(cachePath(region))
	if err != nil {
		return d, err
	}
	err = json.Unmarshal(b, &d)
	return d, err
}
func jsonResponse(w http.ResponseWriter, val any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(val)
}
func markPing() { hadPing.Store(true); lastPing.Store(time.Now().Unix()) }
func playlistURL(chID, remote string) string {
	return "/hls?ch=" + url.QueryEscape(chID) + "&u=" + url.QueryEscape(remote) + "&sig=" + streamSignature(chID, remote)
}
func validateRemote(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil {
		return nil, errors.New("invalid stream address")
	}
	h := strings.ToLower(u.Hostname())
	if h == "localhost" || strings.HasSuffix(h, ".localhost") || strings.HasSuffix(h, ".local") {
		return nil, errors.New("blocked local address")
	}
	if ip := net.ParseIP(h); ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsMulticast()) {
		return nil, errors.New("blocked private address")
	}
	return u, nil
}
func hlsHandler(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("u")
	id := r.URL.Query().Get("ch")
	u, err := validateRemote(raw)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	// A channel ID from the current source is required; arbitrary local HTTP proxying is not supported.
	var found *Channel
	cacheMu.Lock()
	for _, v := range cache {
		for i := range v.data.Channels {
			if v.data.Channels[i].ID == id {
				c := v.data.Channels[i]
				found = &c
				break
			}
		}
		if found != nil {
			break
		}
	}
	cacheMu.Unlock()
	if found == nil {
		http.Error(w, "Refresh channel listings and try again", 404)
		return
	}
	if !permittedStream(*found, raw, r.URL.Query().Get("sig")) {
		http.Error(w, "Stream address not authorised for this channel", 403)
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), "GET", u.String(), nil)
	if err != nil {
		http.Error(w, "Invalid stream", 400)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/124.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	// The third-party index may request a client signature for legacy feeds.
	if r.URL.Query().Get("ua") == "source" {
		req.Header.Set("User-Agent", "otg/1.6 (AppleTv Apple TV 4; tvOS16.0; appletv.client)")
	}
	if h := r.Header.Get("Range"); h != "" {
		req.Header.Set("Range", h)
	}
	resp, err := mediaClient.Do(req)
	if err != nil {
		log.Print("stream request failed (network or provider restriction)")
		http.Error(w, "Stream unavailable (network or provider restriction)", 502)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		http.Error(w, fmt.Sprintf("Stream source returned HTTP %d", resp.StatusCode), 502)
		return
	}
	finalURL := resp.Request.URL
	typ := strings.ToLower(resp.Header.Get("Content-Type"))
	isM3U := strings.HasSuffix(strings.ToLower(finalURL.Path), ".m3u8") || strings.Contains(typ, "mpegurl") || strings.HasSuffix(strings.ToLower(u.Path), ".m3u8")
	if isM3U {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
		if err != nil {
			http.Error(w, "Cannot read stream manifest", 502)
			return
		}
		if !strings.HasPrefix(string(body), "#EXTM3U") {
			http.Error(w, "Provider did not return an HLS playlist", 502)
			return
		}
		lines := strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#EXT-X-KEY") && (strings.Contains(line, "KEYFORMAT=") || strings.Contains(line, "METHOD=SAMPLE-AES")) {
				http.Error(w, "This channel requires a protected player; use the official site", 415)
				return
			}
			if line != "" && !strings.HasPrefix(line, "#") {
				target, er := finalURL.Parse(line)
				if er == nil {
					lines[i] = playlistURL(id, target.String())
				}
			} else if strings.Contains(line, "URI=\"") {
				lines[i] = uriRegexp.ReplaceAllStringFunc(line, func(s string) string {
					parts := uriRegexp.FindStringSubmatch(s)
					if len(parts) < 2 {
						return s
					}
					resolved, er := finalURL.Parse(parts[1])
					if er != nil {
						return s
					}
					return "URI=\"" + playlistURL(id, resolved.String()) + "\""
				})
			}
		}
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = io.WriteString(w, strings.Join(lines, "\n"))
		return
	}
	for _, key := range []string{"Content-Type", "Content-Range", "Accept-Ranges"} {
		if h := resp.Header.Get(key); h != "" {
			w.Header().Set(key, h)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
func heartbeatWatch() {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for range t.C {
		if hadPing.Load() && time.Now().Unix()-lastPing.Load() > 150 {
			log.Print("No player window heartbeat; stopping server")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = server.Shutdown(ctx)
			cancel()
			return
		}
	}
}

// openBrowser always delegates to the OS default URL handler on Windows.
// The GUI and Settings pages both use the same default browser.
func openBrowser(addr string) error {
	if os.Getenv("AUSSIE_TV_NO_BROWSER") == "1" {
		log.Print("Automatic browser launch disabled for testing")
		return nil
	}
	if runtime.GOOS == "windows" {
		if err := openWindowsDefaultBrowser(addr); err != nil {
			return fmt.Errorf("open default Windows browser: %w", err)
		}
		log.Printf("Requested Windows default browser: %s", addr)
		return nil
	}
	for _, name := range []string{"xdg-open", "open"} {
		if bin, err := exec.LookPath(name); err == nil {
			if err := exec.Command(bin, addr).Start(); err == nil {
				return nil
			}
		}
	}
	return fmt.Errorf("no browser opener found; manually visit %s", addr)
}

func prepareLogging() {
	if os.Getenv("AUSSIETV_MODE") == "server" {
		log.SetOutput(os.Stdout)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		return
	}
	path := ""
	if base, err := os.UserCacheDir(); err == nil {
		path = filepath.Join(base, "AussieTV", "startup.log")
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			path = ""
		}
	}
	if path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err == nil {
			if runtime.GOOS == "windows" && os.Getenv("AUSSIETV_DIAGNOSTICS") != "1" {
				log.SetOutput(f)
			} else {
				log.SetOutput(io.MultiWriter(os.Stdout, f))
				fmt.Printf("AussieTV startup log: %s\n", path)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Cannot write log %s: %v\n", path, err)
		}
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
func startupFailure(err error) {
	log.Printf("STARTUP FAILED: %v", err)
	fmt.Fprintln(os.Stderr, "AussieTV couldn't start. The error is above.")
	if runtime.GOOS == "windows" {
		nativeAlert("AussieTV failed to start", err.Error()+"\nSee %LOCALAPPDATA%\\AussieTV\\startup.log")
	}
	os.Exit(1)
}

// serveAppAsset has an allow-list: the local service must not expose arbitrary
// paths from the embedded filesystem or disk. The browser icon works offline.
func serveAppAsset(w http.ResponseWriter, r *http.Request) {
	var path, mime string
	switch r.URL.Path {
	case "/":
		path, mime = "static/index.html", "text/html; charset=utf-8"
	case "/settings":
		path, mime = "static/settings.html", "text/html; charset=utf-8"
	case "/app-icon.svg":
		path, mime = "static/app-icon.svg", "image/svg+xml"
	case "/aussietv-logo.svg":
		path, mime = "static/aussietv-logo.svg", "image/svg+xml"
	case "/favicon-32.png":
		path, mime = "static/favicon-32.png", "image/png"
	case "/favicon-192.png":
		path, mime = "static/favicon-192.png", "image/png"
	case "/favicon.ico":
		path, mime = "static/favicon.ico", "image/x-icon"
	case "/site.webmanifest":
		path, mime = "static/site.webmanifest", "application/manifest+json"
	case "/remote.js":
		path, mime = "static/remote.js", "application/javascript"
	case "/channel-logo-abc-news.svg", "/channel-logo-sbs.svg", "/channel-logo-sbs2.svg",
		"/channel-logo-sbs-world-movies.svg", "/channel-logo-sbs-food.svg", "/channel-logo-nitv.svg":
		path, mime = "static"+r.URL.Path, "image/svg+xml"
	default:
		http.NotFound(w, r)
		return
	}
	body, err := assets.ReadFile(path)
	if err != nil {
		http.Error(w, "Application asset missing", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", mime)
	if path == "static/index.html" || path == "static/settings.html" {
		w.Header().Set("Cache-Control", "no-store")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("X-Content-Type-Options", "nosniff")
	}
	_, _ = w.Write(body)
}

var teletextService *teletext.Service

func initTeletext() error {
	b, err := assets.ReadFile("config/teletext.json")
	if err != nil {
		return err
	}
	configPath := os.Getenv("AUSSIETV_TELETEXT_CONFIG")
	if configPath == "" {
		if exe, e := os.Executable(); e == nil {
			candidate := filepath.Join(filepath.Dir(exe), "teletext.json")
			if _, e := os.Stat(candidate); e == nil {
				configPath = candidate
			}
		}
	}
	if configPath != "" {
		b, err = os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("load teletext page map: %w", err)
		}
	}
	cfg, err := teletext.LoadConfig(b)
	if err != nil {
		return err
	}
	teletextService = teletext.New(cfg, func(region string) []teletext.Channel {
		data := loadChannels(region)
		result := make([]teletext.Channel, 0, len(data.Channels))
		for _, ch := range data.Channels {
			row := teletext.Channel{Name: ch.Name, Network: ch.Network, Number: ch.Number}
			for _, p := range ch.Programs {
				row.Programmes = append(row.Programmes, teletext.Programme{Name: p.Title, Start: time.Unix(p.Start, 0), End: time.Unix(p.End, 0)})
			}
			result = append(result, row)
		}
		return result
	})
	return nil
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--healthcheck" {
		if err := serverHealthcheck(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if code, handled := maybeRunFirewallHelper(os.Args); handled {
		os.Exit(code)
	}
	prepareLogging()
	initArtworkDiskCache()
	if err := initTeletext(); err != nil {
		log.Printf("Teletext unavailable: %v", err)
	}

	serverMode := os.Getenv("AUSSIETV_MODE") == "server"
	webCfg, err := dockerWebConfigFromEnv(serverMode)
	if err != nil {
		startupFailure(err)
	}
	feedCfg, envEnabled, feedErr := jellyfinConfigFromEnv()
	if feedErr != nil {
		startupFailure(feedErr)
	}
	if serverMode && !envEnabled && webCfg == nil {
		startupFailure(errors.New("server mode requires configured Jellyfin feed credentials or authenticated web UI"))
	}
	supervisor := &feedSupervisor{}
	defer supervisor.Close()
	var settings *settingsController
	if !serverMode {
		var err error
		settings, err = newSettingsController(supervisor, envEnabled)
		if err != nil {
			// Never hide the entire TV app because an older/corrupted settings file
			// cannot be loaded. Keep the original file for diagnostics.
			log.Printf("Cannot load saved settings: %v", err)
			settings, err = newDefaultSettingsController(supervisor, envEnabled)
			if err != nil {
				startupFailure(fmt.Errorf("create default settings: %w", err))
			}
		}
	}
	if envEnabled {
		if settings != nil {
			settings.config = desktopSettings{Enabled: true, LAN: strings.HasPrefix(feedCfg.Bind, "0.0.0.0:"), Discovery: feedCfg.Discovery, StreamMode: feedCfg.StreamMode, BaseURL: feedCfg.BaseURL, Region: feedCfg.Region, Key: feedCfg.Key}
		}
		if err := supervisor.Replace(feedCfg, true); err != nil {
			if serverMode {
				startupFailure(fmt.Errorf("Jellyfin feed listener: %w", err))
			}
			log.Printf("Jellyfin feed unavailable (TV interface still starts): %v", err)
		}
	} else if settings != nil {
		cfg, enabled := desktopToJellyfin(settings.config)
		if err := supervisor.Replace(cfg, enabled); err != nil {
			// This is an optional network integration, never a fatal TV startup error.
			log.Printf("Jellyfin feed unavailable (TV interface still starts): %v", err)
		}
	}
	if serverMode && envEnabled {
		warmCtx, stopWarm := context.WithCancel(context.Background())
		defer stopWarm()
		go warmProgrammeArtwork(warmCtx, feedCfg.Region)
	}
	log.Printf("AussieTV v%s starting on %s/%s", version, runtime.GOOS, runtime.GOARCH)
	mux := http.NewServeMux()
	if teletextService != nil {
		mux.Handle("GET /teletext/", teletextService.Handler())
	}
	mux.HandleFunc("GET /", serveAppAsset)
	mux.HandleFunc("GET /api/artwork", artworkHandler)
	mux.HandleFunc("GET /api/logo", logoHandler)
	mux.HandleFunc("GET /api/channels", func(w http.ResponseWriter, r *http.Request) {
		markPing()
		jsonResponse(w, loadChannels(r.URL.Query().Get("region")))
	})
	mux.HandleFunc("POST /api/heartbeat", func(w http.ResponseWriter, r *http.Request) { markPing(); w.WriteHeader(204) })
	mux.HandleFunc("GET /hls", hlsHandler)
	if settings != nil {
		mux.HandleFunc("GET /api/settings", settings.handler)
		mux.HandleFunc("POST /api/settings", settings.handler)
	}
	mux.HandleFunc("GET /channel.m3u", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("ch")
		d := loadChannels(r.URL.Query().Get("region"))
		for _, ch := range d.Channels {
			if ch.ID == id && ch.Stream != "" {
				w.Header().Set("Content-Type", "application/x-mpegurl")
				w.Header().Set("Content-Disposition", "attachment; filename=\"AussieTV-"+strings.ReplaceAll(ch.Name, " ", "-")+".m3u\"")
				fmt.Fprintf(w, "#EXTM3U\n#EXTINF:-1,%s\n%s\n", ch.Name, ch.Stream)
				return
			}
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("POST /api/quit", func(w http.ResponseWriter, r *http.Request) {
		// Browser windows never own the lifetime of a tray application.
		if serverMode || trayReady.Load() {
			http.Error(w, "Use the tray menu to exit", http.StatusForbidden)
			return
		}
		w.WriteHeader(204)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = server.Shutdown(ctx)
		}()
	})
	bind := "127.0.0.1:45781"
	if webCfg != nil {
		bind = webCfg.bind
	}
	listener, err := net.Listen("tcp", bind)
	if err != nil && webCfg == nil {
		log.Printf("Port 45781 unavailable (%v); trying a free port", err)
		listener, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err != nil {
		startupFailure(err)
	}
	server = &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if host != "127.0.0.1" && host != "localhost" {
			http.Error(w, "Invalid local host", 403)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || u.Host != r.Host || u.Scheme != "http" {
				http.Error(w, "Invalid origin", 403)
				return
			}
		}
		mux.ServeHTTP(w, r)
	}), ReadHeaderTimeout: 10 * time.Second}
	if webCfg != nil {
		server.Handler = webCfg.handler(mux)
		server.IdleTimeout = 60 * time.Second
		server.MaxHeaderBytes = 1 << 16
	}
	addr := "http://" + listener.Addr().String()
	log.Printf("AussieTV listening at %s", addr)
	if !serverMode {
		if runtime.GOOS == "windows" && os.Getenv("AUSSIETV_NO_TRAY") != "1" {
			logFile := filepath.Join(os.Getenv("LOCALAPPDATA"), "AussieTV", "startup.log")
			stop := func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_ = server.Shutdown(ctx)
			}
			if err := startTray(addr, addr+"/settings", logFile, stop); err != nil {
				log.Printf("Tray registration failed; continuing in browser mode: %v", err)
				go heartbeatWatch()
			} else {
				trayReady.Store(true)
			}
		} else {
			go heartbeatWatch()
		}
		go func() {
			time.Sleep(200 * time.Millisecond)
			if err := openBrowser(addr); err != nil {
				log.Printf("Could not open browser: %v", err)
			}
		}()
	}
	var shutdownDone chan struct{}
	var signalCtx context.Context
	if serverMode {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		signalCtx = ctx
		shutdownDone = make(chan struct{})
		go func() {
			defer close(shutdownDone)
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				_ = server.Close()
			}
		}()
	}
	err = server.Serve(listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		startupFailure(fmt.Errorf("server stopped: %w", err))
	}
	if signalCtx != nil && signalCtx.Err() != nil {
		<-shutdownDone
	}
	log.Println("AussieTV stopped")
}
