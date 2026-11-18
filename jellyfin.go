package main

// Jellyfin's built-in M3U/XMLTV integration is preferable to pretending to be
// an HDHomeRun device. This server exposes only read-only IPTV feed endpoints.
// It is intentionally separate from the desktop HTTP server and its quit API.

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type jellyfinConfig struct {
	Bind    string
	BaseURL string
	Key     string
	Region  string
}

func jellyfinConfigFromEnv() (jellyfinConfig, bool, error) {
	bind := strings.TrimSpace(os.Getenv("AUSSIETV_JELLYFIN_BIND"))
	if bind == "" {
		return jellyfinConfig{}, false, nil
	}
	cfg := jellyfinConfig{Bind: bind, BaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("AUSSIETV_JELLYFIN_BASE_URL")), "/"), Key: strings.TrimSpace(os.Getenv("AUSSIETV_JELLYFIN_KEY")), Region: safeRegion(strings.TrimSpace(os.Getenv("AUSSIETV_JELLYFIN_REGION")))}
	if len(cfg.Key) < 24 {
		return cfg, true, errors.New("AUSSIETV_JELLYFIN_KEY must contain at least 24 characters")
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return cfg, true, errors.New("AUSSIETV_JELLYFIN_BASE_URL must be an http(s) address without path, credentials or query")
	}
	if _, err := url.Parse("http://" + cfg.Bind); err != nil {
		return cfg, true, fmt.Errorf("invalid AUSSIETV_JELLYFIN_BIND: %w", err)
	}
	return cfg, true, nil
}

func xmlTVID(region string, ch Channel) string {
	return "aussietv." + strings.ToLower(region) + "." + base64.RawURLEncoding.EncodeToString([]byte(ch.ID))
}
func feedChannels(data ChannelData) []Channel {
	var channels []Channel
	for _, ch := range data.Channels {
		if ch.ID == "" || ch.Stream == "" {
			continue
		}
		if _, err := validateRemote(ch.Stream); err != nil {
			continue
		}
		channels = append(channels, ch)
	}
	return channels
}
func singleLine(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == '"' || r < 32 || r == 127 {
			return ' '
		}
		return r
	}, value)
}
func xmltvTime(sec int64) string { return time.Unix(sec, 0).UTC().Format("20060102150405 -0000") }
func m3uFeed(channels []Channel, cfg jellyfinConfig) string {
	var w strings.Builder
	w.WriteString("#EXTM3U\n")
	for _, ch := range channels {
		id := xmlTVID(cfg.Region, ch)
		logo := ""
		if _, err := validateRemote(ch.Logo); err == nil {
			logo = singleLine(ch.Logo)
		}
		fmt.Fprintf(&w, "#EXTINF:-1 tvg-id=\"%s\" tvg-name=\"%s\" tvg-chno=\"%s\" tvg-logo=\"%s\" group-title=\"%s\",%s\n",
			singleLine(id), singleLine(ch.Name), singleLine(ch.Number), logo, singleLine(ch.Network), singleLine(ch.Name))
		fmt.Fprintf(&w, "%s/jellyfin/stream/%s?key=%s\n", cfg.BaseURL, base64.RawURLEncoding.EncodeToString([]byte(ch.ID)), url.QueryEscape(cfg.Key))
	}
	return w.String()
}

type xmltvName struct {
	Lang  string `xml:"lang,attr,omitempty"`
	Value string `xml:",chardata"`
}
type xmltvIcon struct {
	Src string `xml:"src,attr"`
}
type xmltvChannel struct {
	ID    string      `xml:"id,attr"`
	Names []xmltvName `xml:"display-name"`
	Icon  *xmltvIcon  `xml:"icon,omitempty"`
}
type xmltvProgramme struct {
	Start   string    `xml:"start,attr"`
	Stop    string    `xml:"stop,attr"`
	Channel string    `xml:"channel,attr"`
	Title   xmltvName `xml:"title"`
}
type xmltvDocument struct {
	XMLName    xml.Name         `xml:"tv"`
	Generator  string           `xml:"generator-info-name,attr"`
	Channels   []xmltvChannel   `xml:"channel"`
	Programmes []xmltvProgramme `xml:"programme"`
}

func xmltvFeed(channels []Channel, region string) ([]byte, error) {
	doc := xmltvDocument{Generator: "AussieTV", Channels: []xmltvChannel{}, Programmes: []xmltvProgramme{}}
	for _, ch := range channels {
		id := xmlTVID(region, ch)
		row := xmltvChannel{ID: id, Names: []xmltvName{{Value: ch.Name}, {Value: ch.Number}}}
		if _, err := validateRemote(ch.Logo); err == nil {
			row.Icon = &xmltvIcon{Src: ch.Logo}
		}
		doc.Channels = append(doc.Channels, row)
		for _, p := range ch.Programs {
			if p.Start <= 0 || p.End <= p.Start || p.Title == "" {
				continue
			}
			doc.Programmes = append(doc.Programmes, xmltvProgramme{Start: xmltvTime(p.Start), Stop: xmltvTime(p.End), Channel: id, Title: xmltvName{Lang: "en", Value: p.Title}})
		}
	}
	bytes, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), append(bytes, '\n')...), nil
}

// newJellyfinHandler has its own mux with no desktop APIs. It never accepts
// arbitrary URLs to fetch, preventing it from becoming a public proxy.
func newJellyfinHandler(cfg jellyfinConfig, load func(string) ChannelData) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /jellyfin/channels.m3u", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/x-mpegurl; charset=utf-8")
		_, _ = io.WriteString(w, m3uFeed(feedChannels(load(cfg.Region)), cfg))
	})
	mux.HandleFunc("GET /jellyfin/guide.xml", func(w http.ResponseWriter, r *http.Request) {
		content, err := xmltvFeed(feedChannels(load(cfg.Region)), cfg.Region)
		if err != nil {
			http.Error(w, "guide encoding error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write(content)
	})
	mux.HandleFunc("GET /jellyfin/stream/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := base64.RawURLEncoding.DecodeString(r.PathValue("id"))
		if err != nil || len(id) > 512 {
			http.NotFound(w, r)
			return
		}
		for _, ch := range feedChannels(load(cfg.Region)) {
			if ch.ID == string(id) {
				// The upstream HLS service handles its own manifests and segments.
				// Some feeds may require headers or DRM this redirect cannot supply.
				http.Redirect(w, r, ch.Stream, http.StatusTemporaryRedirect)
				return
			}
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("GET /jellyfin/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "ok\n")
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		// A key must be present even for health checks; this is a LAN service.
		provided := sha256.Sum256([]byte(r.URL.Query().Get("key")))
		expected := sha256.Sum256([]byte(cfg.Key))
		if subtle.ConstantTimeCompare(provided[:], expected[:]) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		mux.ServeHTTP(w, r)
	})
}
