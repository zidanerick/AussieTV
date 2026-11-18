package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Settings are local to the Windows account, never stored in the public web root.
// Do not log the Jellyfin key or its playlist/guide URLs.
type desktopSettings struct {
	Enabled    bool   `json:"enabled"`
	LAN        bool   `json:"lan"`
	Discovery  bool   `json:"discovery"`
	BaseURL    string `json:"baseUrl"`
	Region     string `json:"region"`
	Key        string `json:"key"`
	StreamMode string `json:"streamMode,omitempty"`
}
type publicSettings struct {
	Enabled          bool   `json:"enabled"`
	LAN              bool   `json:"lan"`
	Discovery        bool   `json:"discovery"`
	StreamMode       string `json:"streamMode"`
	BaseURL          string `json:"baseUrl"`
	Region           string `json:"region"`
	DefaultLANURL    string `json:"defaultLanUrl"`
	M3U              string `json:"m3u"`
	XMLTV            string `json:"xmltv"`
	DiagnosticsURL   string `json:"diagnosticsUrl"`
	Status           string `json:"status"`
	Managed          bool   `json:"managed"`
	CSRF             string `json:"csrf"`
	FirewallMessage  string `json:"firewallMessage,omitempty"`
	FFmpegStatus     string `json:"ffmpegStatus,omitempty"`
	DiscoveryProbes  uint64 `json:"discoveryProbes"`
	DiscoveryReplies uint64 `json:"discoveryReplies"`
	DiscoveryHTTP    uint64 `json:"discoveryHTTP"`
}

func settingsFile() string {
	root := os.Getenv("LOCALAPPDATA")
	if root == "" {
		root, _ = os.UserConfigDir()
	}
	if root == "" {
		root = os.TempDir()
	}
	return filepath.Join(root, "AussieTV", "settings.json")
}
func firstLANIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addresses {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			v := ip.To4()
			if v != nil && v.IsPrivate() {
				return v.String()
			}
		}
	}
	return ""
}
func defaultSettings() desktopSettings {
	return desktopSettings{Region: "Melbourne", BaseURL: "http://127.0.0.1:45782"}
}
func readSettings() (desktopSettings, error) {
	cfg := defaultSettings()
	b, err := os.ReadFile(settingsFile())
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err = json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if err = validateSettings(cfg); err != nil {
		return cfg, fmt.Errorf("invalid saved settings: %w", err)
	}
	return cfg, nil
}
func validateSettings(cfg desktopSettings) error {
	if !validStreamMode(cfg.StreamMode) {
		return errors.New("Jellyfin stream mode must be redirect, relay, or mpegts")
	}
	if !regionAllowed[cfg.Region] {
		return errors.New("select a supported Australian TV region")
	}
	if len(cfg.Key) > 0 && len(cfg.Key) < 24 {
		return errors.New("invalid Jellyfin key")
	}
	if strings.ContainsAny(cfg.BaseURL, "\r\n\t") {
		return errors.New("invalid Jellyfin address")
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Port() != "45782" {
		return errors.New("use a complete http(s) address ending in :45782, without a path or query")
	}
	if cfg.LAN {
		if u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" {
			return errors.New("for LAN access, enter this PC's reachable LAN address")
		}
	}
	if cfg.Discovery {
		if !cfg.Enabled || !cfg.LAN {
			return errors.New("Enable LAN Jellyfin feed before HDHomeRun discovery")
		}
		if _, err := hdhrHTTPBind(jellyfinConfig{BaseURL: cfg.BaseURL}); err != nil {
			return err
		}
	}
	if cfg.Enabled && len(cfg.Key) < 24 {
		return errors.New("missing Jellyfin access key")
	}
	return nil
}
func newSettingsKey() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
func storeSettings(cfg desktopSettings) error {
	if err := validateSettings(cfg); err != nil {
		return err
	}
	p := settingsFile()
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	// Windows user-profile ACL limits access; keep Unix mode restricted as well.
	temp, err := os.CreateTemp(filepath.Dir(p), ".settings-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err = temp.Chmod(0600); err != nil {
		temp.Close()
		return err
	}
	if _, err = temp.Write(append(data, '\n')); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	// os.Rename replaces the old file on standard Windows NTFS volumes.
	return os.Rename(temp.Name(), p)
}
func desktopToJellyfin(cfg desktopSettings) (jellyfinConfig, bool) {
	if !cfg.Enabled {
		return jellyfinConfig{}, false
	}
	bind := "127.0.0.1:45782"
	if cfg.LAN {
		bind = "0.0.0.0:45782"
	}
	return jellyfinConfig{Bind: bind, BaseURL: cfg.BaseURL, Region: cfg.Region, Key: cfg.Key, Discovery: cfg.Discovery, StreamMode: cfg.StreamMode}, true
}

// feedSupervisor protects the existing, read-only and authenticated Jellyfin
// listener. Settings changes replace only the feed listener, never the desktop
// server or its private /api/quit handler.
type feedSupervisor struct {
	mu            sync.Mutex
	server        *http.Server
	listener      net.Listener
	discoveryConn *net.UDPConn
	hdhrServer    *http.Server
	hdhrListener  net.Listener
	discoveryNote string
	config        jellyfinConfig
	enabled       bool
	lastError     string
}

func (s *feedSupervisor) stopLocked() {
	if s.server != nil {
		s.server.Close()
	}
	if s.listener != nil {
		s.listener.Close()
	}
	if s.discoveryConn != nil {
		s.discoveryConn.Close()
	}
	if s.hdhrServer != nil {
		s.hdhrServer.Close()
	}
	if s.hdhrListener != nil {
		s.hdhrListener.Close()
	}
	s.hdhrServer = nil
	s.hdhrListener = nil
	s.discoveryNote = ""
	discoveryUDPListening.Store(false)
	discoveryHTTPListening.Store(false)
	s.server = nil
	s.listener = nil
	s.discoveryConn = nil
	s.enabled = false
}

// openHDHRListeners keeps UDP 65001 independent of optional HTTP port 80.
// Injected listeners make port conflicts deterministic in regression tests.
func openHDHRListeners(cfg jellyfinConfig,
	openTCP func(string, string) (net.Listener, error),
	openUDP func(string, *net.UDPAddr) (*net.UDPConn, error),
) (*net.UDPConn, net.Listener, string) {
	// Invalid/non-LAN advertised addresses are still fail-closed. Only a
	// valid LAN configuration is allowed to start an unauthenticated UDP
	// discovery listener, even if HTTP port 80 happens to be unavailable.
	bind, err := hdhrHTTPBind(cfg)
	if err != nil {
		return nil, nil, fmt.Sprintf("Auto-detection unavailable: %v", err)
	}
	var notes []string
	udp, udpErr := openUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: hdhrPort})
	if udpErr != nil {
		notes = append(notes, fmt.Sprintf("UDP %d not listening: %v", hdhrPort, udpErr))
	}
	listener, err := openTCP("tcp", bind)
	if err != nil {
		notes = append(notes, fmt.Sprintf("TCP 80 not listening: %v", err))
	}
	return udp, listener, strings.Join(notes, "; ")
}

func (s *feedSupervisor) startLocked(cfg jellyfinConfig, enabled bool) error {
	if !enabled {
		return nil
	}
	ln, err := net.Listen("tcp", cfg.Bind)
	if err != nil {
		return err
	}
	var discoverConn *net.UDPConn
	var hdhrListener net.Listener
	var hdhrServer *http.Server
	var discoveryNote string
	if cfg.Discovery {
		// Probe reception must not depend on TCP 80 being free. On Windows
		// HTTP.sys or another app may own port 80 while UDP discovery is
		// perfectly functional. Count probes even in this partial state.
		discoverConn, hdhrListener, discoveryNote = openHDHRListeners(cfg, net.Listen, net.ListenUDP)
		if hdhrListener != nil {
			hdhrServer = &http.Server{Handler: hdhrHTTP(cfg, loadChannels), ReadHeaderTimeout: 10 * time.Second, MaxHeaderBytes: 1 << 16}
		}
	}
	discoveryUDPListening.Store(discoverConn != nil)
	discoveryHTTPListening.Store(hdhrListener != nil)
	srv := &http.Server{Handler: newJellyfinHandler(cfg, loadChannels), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, WriteTimeout: 45 * time.Second, MaxHeaderBytes: 1 << 16}
	s.listener = ln
	s.server = srv
	s.discoveryConn = discoverConn
	s.hdhrServer = hdhrServer
	s.hdhrListener = hdhrListener
	s.discoveryNote = discoveryNote
	s.config = cfg
	s.enabled = true
	if discoverConn != nil {
		go runHDHRDiscovery(discoverConn, cfg)
	}
	if hdhrServer != nil {
		go func() {
			if err := hdhrServer.Serve(hdhrListener); err != nil && err != http.ErrServerClosed {
				log.Printf("HDHomeRun discovery HTTP listener stopped: %v", err)
			}
		}()
	}
	if discoveryNote != "" {
		log.Print(discoveryNote)
	}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("Jellyfin feed stopped: %v", err)
		}
	}()
	log.Printf("Jellyfin feed listening on %s (%s)", ln.Addr().String(), cfg.Region)
	return nil
}
func (s *feedSupervisor) Replace(cfg jellyfinConfig, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, wasEnabled := s.config, s.enabled
	s.stopLocked()
	if err := s.startLocked(cfg, enabled); err != nil {
		s.lastError = err.Error()
		if rollback := s.startLocked(old, wasEnabled); rollback != nil {
			log.Printf("Jellyfin feed rollback failed: %v", rollback)
		}
		return err
	}
	s.lastError = ""
	return nil
}
func (s *feedSupervisor) Close() { s.mu.Lock(); defer s.mu.Unlock(); s.stopLocked() }
func (s *feedSupervisor) Status() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.enabled {
		if s.lastError != "" {
			return "Unavailable: " + s.lastError
		}
		return "Disabled"
	}
	status := "Listening on " + s.listener.Addr().String()
	if s.discoveryNote != "" {
		return status + "; " + s.discoveryNote
	}
	if s.discoveryConn != nil && s.hdhrListener != nil {
		return status + "; HDHomeRun discovery active (UDP 65001 + HTTP 80)"
	}
	if s.discoveryConn != nil {
		return status + "; HDHomeRun UDP 65001 listening (HTTP 80 unavailable)"
	}
	if s.hdhrListener != nil {
		return status + "; HDHomeRun HTTP 80 listening (UDP 65001 unavailable)"
	}
	return status
}

type settingsController struct {
	mu              sync.Mutex
	config          desktopSettings
	managed         bool
	csrf            string
	feed            *feedSupervisor
	firewallMessage string
}

func newDefaultSettingsController(feed *feedSupervisor, managed bool) (*settingsController, error) {
	key, err := newSettingsKey()
	if err != nil {
		return nil, err
	}
	return &settingsController{config: defaultSettings(), managed: managed, csrf: key, feed: feed}, nil
}

func newSettingsController(feed *feedSupervisor, managed bool) (*settingsController, error) {
	key, err := newSettingsKey()
	if err != nil {
		return nil, err
	}
	cfg, err := readSettings()
	if err != nil {
		return nil, err
	}
	return &settingsController{config: cfg, managed: managed, csrf: key, feed: feed}, nil
}
func (c *settingsController) present() publicSettings {
	cfg := c.config
	p := publicSettings{Enabled: cfg.Enabled, LAN: cfg.LAN, Discovery: cfg.Discovery, StreamMode: effectiveStreamMode(cfg.StreamMode), BaseURL: cfg.BaseURL, Region: cfg.Region, Managed: c.managed, CSRF: c.csrf, Status: c.feed.Status(), FirewallMessage: c.firewallMessage}
	if effectiveStreamMode(cfg.StreamMode) == "mpegts" {
		available, runnable, source := ffmpegHealth()
		switch {
		case !available:
			p.FFmpegStatus = "NOT FOUND: extract the complete ZIP including ffmpeg/ffmpeg.exe"
		case !runnable:
			p.FFmpegStatus = "FOUND (" + source + ") but failed to execute; Windows security or missing dependencies may be blocking it"
		default:
			p.FFmpegStatus = "Ready (" + source + ")"
		}
	}
	p.DiscoveryProbes = discoveryProbes.Load()
	p.DiscoveryReplies = discoveryReplies.Load()
	p.DiscoveryHTTP = discoveryHTTPRequests.Load()
	if ip := firstLANIPv4(); ip != "" {
		p.DefaultLANURL = "http://" + net.JoinHostPort(ip, "45782")
	}
	if cfg.Enabled && !c.managed {
		p.M3U = cfg.BaseURL + "/jellyfin/channels.m3u?key=" + url.QueryEscape(cfg.Key)
		p.XMLTV = cfg.BaseURL + "/jellyfin/guide.xml?key=" + url.QueryEscape(cfg.Key)
		p.DiagnosticsURL = cfg.BaseURL + "/jellyfin/diagnostics?key=" + url.QueryEscape(cfg.Key)
	}
	return p
}
func (c *settingsController) handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method == http.MethodGet {
		c.mu.Lock()
		p := c.present()
		c.mu.Unlock()
		jsonResponse(w, p)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", 405)
		return
	}
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		http.Error(w, "JSON required", 415)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-AussieTV-CSRF")), []byte(c.csrf)) != 1 {
		http.Error(w, "Forbidden", 403)
		return
	}
	if c.managed {
		http.Error(w, "Jellyfin feed is controlled by environment variables", 409)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var request struct {
		Enabled        bool   `json:"enabled"`
		LAN            bool   `json:"lan"`
		Discovery      bool   `json:"discovery"`
		StreamMode     string `json:"streamMode"`
		BaseURL        string `json:"baseUrl"`
		Region         string `json:"region"`
		Rotate         bool   `json:"rotate"`
		FirewallRepair bool   `json:"firewallRepair"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&request); err != nil {
		http.Error(w, "Invalid settings", 400)
		return
	}
	cfg := c.config
	cfg.Enabled = request.Enabled
	cfg.LAN = request.LAN
	cfg.Discovery = request.Discovery
	cfg.StreamMode = request.StreamMode
	cfg.BaseURL = strings.TrimSpace(request.BaseURL)
	cfg.Region = request.Region
	if cfg.Key == "" || request.Rotate {
		var err error
		cfg.Key, err = newSettingsKey()
		if err != nil {
			http.Error(w, "Cannot generate key", 500)
			return
		}
	}
	if err := validateSettings(cfg); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	old := c.config
	next, on := desktopToJellyfin(cfg)
	if err := c.feed.Replace(next, on); err != nil {
		http.Error(w, "Cannot activate feed: "+err.Error(), 409)
		return
	}
	if err := storeSettings(cfg); err != nil {
		previous, priorOn := desktopToJellyfin(old)
		_ = c.feed.Replace(previous, priorOn)
		http.Error(w, "Settings cannot be saved: "+err.Error(), 500)
		return
	}
	c.config = cfg
	c.firewallMessage = ""
	if err := updateFirewall(old, cfg, request.FirewallRepair); err != nil {
		// Do not re-enable a feed that the user just disabled when UAC is declined.
		// The new settings are already persisted; make the failed firewall step
		// visible and provide a one-click Repair action in Settings.
		log.Printf("Windows Firewall change incomplete: %v", err)
		c.firewallMessage = "Saved, but Windows Firewall could not be updated: " + err.Error() + ". Use Repair firewall access after approving the Windows prompt."
	} else if firewallFor(old) != firewallFor(cfg) || request.FirewallRepair {
		if cfg.Enabled && cfg.LAN {
			c.firewallMessage = "Windows Firewall configured for Private networks and local-subnet devices only."
		} else {
			c.firewallMessage = "AussieTV's Private-network firewall rules were removed (local-only use needs none)."
		}
	}
	jsonResponse(w, c.present())
}
