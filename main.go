package main

import (
	"context"
	"embed"
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
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//go:embed static/*
var assets embed.FS

const sourceBase = "https://i.mjh.nz/au/"

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
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Number   string    `json:"number"`
	Network  string    `json:"network"`
	Logo     string    `json:"logo,omitempty"`
	Stream   string    `json:"stream,omitempty"`
	Official string    `json:"official"`
	Programs []Program `json:"programs"`
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
var mediaClient = &http.Client{Timeout: 0, Transport: &http.Transport{
	Proxy: http.ProxyFromEnvironment, MaxIdleConns: 80, MaxIdleConnsPerHost: 24,
	IdleConnTimeout: 45 * time.Second, TLSHandshakeTimeout: 12 * time.Second, ResponseHeaderTimeout: 25 * time.Second,
}}

type cacheEntry struct {
	data ChannelData
	at   time.Time
}

var cacheMu sync.Mutex
var cache = map[string]cacheEntry{}
var lastPing atomic.Int64
var hadPing atomic.Bool
var server *http.Server
var uriRegexp = regexp.MustCompile(`URI="([^"]+)"`)

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
	cacheMu.Unlock()
	if ok && time.Since(old.at) < 20*time.Minute {
		return old.data
	}
	req, _ := http.NewRequest("GET", sourceBase+region+"/tv.json", nil)
	req.Header.Set("User-Agent", "AussieTV/1.0 (+personal local player)")
	response, err := client.Do(req)
	if err == nil && response.StatusCode != 200 {
		err = fmt.Errorf("EPG server HTTP %d", response.StatusCode)
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
	if ok {
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
		ch := Channel{ID: id, Name: entry.Name, Number: spec.number, Network: entry.Network, Logo: entry.Logo, Stream: entry.Master, Official: official[spec.net], Programs: []Program{}}
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
	return "/hls?ch=" + url.QueryEscape(chID) + "&u=" + url.QueryEscape(remote)
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
	region := safeRegion(r.URL.Query().Get("region"))
	_ = region
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
	req, err := http.NewRequestWithContext(r.Context(), "GET", u.String(), nil)
	if err != nil {
		http.Error(w, "Invalid stream", 400)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/124.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	// The third-party index may request a client signature for legacy feeds.
	if r.URL.Query().Get("ua") == "source" {
		req.Header.Set("User-Agent", "otg/1.5.1 (AppleTv Apple TV 4; tvOS16.0; appletv.client)")
	}
	if h := r.Header.Get("Range"); h != "" {
		req.Header.Set("Range", h)
	}
	resp, err := mediaClient.Do(req)
	if err != nil {
		log.Printf("stream error %s: %v", id, err)
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
func openBrowser(addr string) {
	if os.Getenv("AUSSIE_TV_NO_BROWSER") == "1" {
		return
	}
	if runtime.GOOS == "windows" {
		pf := os.Getenv("ProgramFiles")
		pfx := os.Getenv("ProgramFiles(x86)")
		local := os.Getenv("LOCALAPPDATA")
		paths := []string{filepath.Join(pfx, "Microsoft", "Edge", "Application", "msedge.exe"), filepath.Join(pf, "Microsoft", "Edge", "Application", "msedge.exe"), filepath.Join(pf, "Google", "Chrome", "Application", "chrome.exe"), filepath.Join(pfx, "Google", "Chrome", "Application", "chrome.exe"), filepath.Join(local, "Google", "Chrome", "Application", "chrome.exe")}
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				_ = exec.Command(path, "--app="+addr, "--window-size=1360,900").Start()
				return
			}
		}
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", addr).Start()
		return
	}
	for _, name := range []string{"xdg-open", "open"} {
		if p, err := exec.LookPath(name); err == nil {
			_ = exec.Command(p, addr).Start()
			return
		}
	}
	fmt.Println("Open this address:", addr)
}
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		b, err := assets.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, "UI missing", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("GET /api/channels", func(w http.ResponseWriter, r *http.Request) {
		markPing()
		jsonResponse(w, loadChannels(r.URL.Query().Get("region")))
	})
	mux.HandleFunc("POST /api/heartbeat", func(w http.ResponseWriter, r *http.Request) { markPing(); w.WriteHeader(204) })
	mux.HandleFunc("GET /hls", hlsHandler)
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
		markPing()
		w.WriteHeader(204)
		go func() {
			time.Sleep(400 * time.Millisecond)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = server.Shutdown(ctx)
		}()
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	server = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	addr := "http://" + listener.Addr().String()
	go heartbeatWatch()
	go func() { time.Sleep(450 * time.Millisecond); openBrowser(addr) }()
	log.Println("Aussie TV listening on", addr)
	err = server.Serve(listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("Server failed: %v", err)
	}
}
