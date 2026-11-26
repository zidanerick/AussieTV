package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// Discord has a separate read-only listener. It never registers desktop
// settings, feed, quit, artwork, logo or streaming routes.
type discordChannel struct {
	Name string `json:"name"`
	Number string `json:"number"`
	Network string `json:"network"`
	Official string `json:"official"`
	Programs []Program `json:"programs"`
}

type discordGuide struct {
	Region string `json:"region"`
	Updated string `json:"updated"`
	Warning string `json:"warning,omitempty"`
	Channels []discordChannel `json:"channels"`
}

type discordGuideSlot struct {
	mu sync.Mutex
	at time.Time
	data discordGuide
}

var discordClientID = regexp.MustCompile(`^[0-9]{17,22}$`)

func discordActivityHandler(web fs.FS, appID string, load func(string) ChannelData) http.Handler {
	slots := make(map[string]*discordGuideSlot, len(regionAllowed))
	for region := range regionAllowed {
		slots[region] = &discordGuideSlot{}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"applicationId": appID})
	})
	mux.HandleFunc("GET /api/guide", func(w http.ResponseWriter, r *http.Request) {
		region := r.URL.Query().Get("region")
		if region == "" { region = "Melbourne" }
		slot, ok := slots[region]
		if !ok { http.Error(w, "Unknown region", http.StatusBadRequest); return }
		// Bound upstream refresh concurrency and negative-cache failures too.
		slot.mu.Lock()
		if slot.at.IsZero() || time.Since(slot.at) >= time.Minute {
			source := load(region)
			out := discordGuide{Region: region, Updated: source.Updated, Warning: source.Warning, Channels: []discordChannel{}}
			for _, ch := range source.Channels {
				link, known := official[ch.Network]
				if !known { continue }
				out.Channels = append(out.Channels, discordChannel{
					Name: ch.Name, Number: ch.Number, Network: ch.Network,
					Official: link, Programs: ch.Programs,
				})
			}
			slot.data, slot.at = out, time.Now()
		}
		data := slot.data
		slot.mu.Unlock()
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(data)
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var name, mime string
		switch r.URL.Path {
		case "/": name, mime = "index.html", "text/html; charset=utf-8"
		case "/app.js": name, mime = "app.js", "text/javascript; charset=utf-8"
		case "/style.css": name, mime = "style.css", "text/css; charset=utf-8"
		default: http.NotFound(w, r); return
		}
		body, err := fs.ReadFile(web, name)
		if err != nil { http.NotFound(w, r); return }
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", mime)
		_, _ = w.Write(body)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors https://discord.com https://*.discord.com https://*.discordsays.com")
		// Discord's Activity proxy can forward requests with this prefix.
		if r.URL.Path == "/.proxy" || len(r.URL.Path) >= 8 && r.URL.Path[:8] == "/.proxy/" {
			r = r.Clone(r.Context())
			r.URL.Path = r.URL.Path[7:]
			if r.URL.Path == "" { r.URL.Path = "/" }
			r.URL.RawPath = ""
		}
		mux.ServeHTTP(w, r)
	})
}

func runDiscordActivity() error {
	appID := os.Getenv("AUSSIETV_DISCORD_APP_ID")
	if !discordClientID.MatchString(appID) {
		return errors.New("set AUSSIETV_DISCORD_APP_ID to your Discord application ID")
	}
	addr := os.Getenv("AUSSIETV_DISCORD_BIND")
	if addr == "" { addr = "127.0.0.1:45783" }
	if _, _, err := net.SplitHostPort(addr); err != nil { return err }
	exe, err := os.Executable()
	if err != nil { return err }
	web := os.DirFS(filepath.Join(filepath.Dir(exe), "discord-web"))
	for _, name := range []string{"index.html", "app.js", "style.css"} {
		if _, err := fs.Stat(web, name); err != nil { return errors.New("missing discord-web build beside executable: " + name) }
	}
	srv := &http.Server{
		Addr: addr, Handler: discordActivityHandler(web, appID, loadChannels),
		ReadHeaderTimeout: 5*time.Second, ReadTimeout: 10*time.Second,
		WriteTimeout: 35*time.Second, IdleTimeout: 60*time.Second,
		MaxHeaderBytes: 16<<10,
	}
	return srv.ListenAndServe()
}
