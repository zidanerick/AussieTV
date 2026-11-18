package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
)

type relayFakeTransport struct {
	mu     sync.Mutex
	paths  []string
	agents []string
}

func (f *relayFakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.paths = append(f.paths, req.URL.Path)
	f.agents = append(f.agents, req.UserAgent())
	f.mu.Unlock()
	var content, mime string
	switch req.URL.Path {
	case "/live/index.m3u8":
		content = "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=800000\nvariant/media.m3u8\n"
		mime = "application/vnd.apple.mpegurl"
	case "/live/variant/media.m3u8":
		content = "#EXTM3U\n#EXT-X-TARGETDURATION:6\n#EXT-X-MAP:URI=\"init.mp4\"\n#EXTINF:6,\nseg0.m4s\n"
		mime = "application/vnd.apple.mpegurl"
	case "/live/variant/seg0.m4s":
		content = "fake-media-segment"
		mime = "video/iso.segment"
	case "/live/variant/init.mp4":
		content = "fake-init"
		mime = "video/mp4"
	default:
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("missing")), Header: make(http.Header), Request: req}, nil
	}
	headers := make(http.Header)
	headers.Set("Content-Type", mime)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(content)), Header: headers, Request: req}, nil
}

func TestJellyfinRelayRewritesManifestsAndSegments(t *testing.T) {
	cfg := exampleCfg()
	cfg.StreamMode = "relay"
	cfg.BaseURL = "http://192.168.1.50:45782"
	ch := sampleFeed().Channels[0]
	ch.Stream = "https://streams.example.org/live/index.m3u8"
	fake := &relayFakeTransport{}
	old := mediaClient
	mediaClient = &http.Client{Transport: fake}
	t.Cleanup(func() { mediaClient = old })
	h := newJellyfinHandler(cfg, func(string) ChannelData { return ChannelData{Region: cfg.Region, Channels: []Channel{ch}} })
	root := "/jellyfin/stream/" + base64.RawURLEncoding.EncodeToString([]byte(ch.ID)) + ".m3u8?key=" + url.QueryEscape(cfg.Key)
	first := requestJellyfin(t, h, root, "GET")
	if first.Code != 200 || !strings.Contains(first.Body.String(), "/jellyfin/media/") || strings.Contains(first.Body.String(), "variant/media.m3u8\n") {
		t.Fatalf("playlist not rewritten: %d %s", first.Code, first.Body.String())
	}
	next := strings.TrimSpace(strings.Split(first.Body.String(), "\n")[2])
	u, err := url.Parse(next)
	if err != nil {
		t.Fatal(err)
	}
	second := requestJellyfin(t, h, u.RequestURI(), "GET")
	if second.Code != 200 || !strings.Contains(second.Body.String(), "/jellyfin/media/") || !strings.Contains(second.Body.String(), "#EXT-X-MAP:URI=\"") {
		t.Fatalf("variant not rewritten: %d %s", second.Code, second.Body.String())
	}
	var segment string
	for _, line := range strings.Split(second.Body.String(), "\n") {
		if strings.HasPrefix(line, "http://") {
			segment = line
		}
	}
	if segment == "" {
		t.Fatal("segment url missing")
	}
	su, _ := url.Parse(segment)
	third := requestJellyfin(t, h, su.RequestURI(), "GET")
	if third.Code != 200 || third.Body.String() != "fake-media-segment" {
		t.Fatalf("segment failed: %d %s", third.Code, third.Body.String())
	}
	// Do not let an attacker reuse the local authenticated feed to request an arbitrary origin.
	q := su.Query()
	q.Set("u", "http://127.0.0.1/private")
	su.RawQuery = q.Encode()
	bad := requestJellyfin(t, h, su.RequestURI(), "GET")
	if bad.Code != 403 {
		t.Fatalf("private target allowed: %d", bad.Code)
	}
	q.Set("u", "https://unrelated.example.org/payload")
	su.RawQuery = q.Encode()
	bad = requestJellyfin(t, h, su.RequestURI(), "GET")
	if bad.Code != 403 {
		t.Fatalf("forged remote allowed: %d", bad.Code)
	}
	noAuth := requestJellyfin(t, h, strings.Split(next, "?")[0], "GET")
	if noAuth.Code != 401 {
		t.Fatalf("unauthenticated relay allowed: %d", noAuth.Code)
	}
}
func TestJellyfinDefaultRedirectRemainsWorking(t *testing.T) {
	cfg := exampleCfg()
	cfg.StreamMode = ""
	ch := sampleFeed().Channels[0]
	h := newJellyfinHandler(cfg, func(string) ChannelData { return ChannelData{Channels: []Channel{ch}} })
	root := "/jellyfin/stream/" + base64.RawURLEncoding.EncodeToString([]byte(ch.ID)) + ".m3u8?key=" + url.QueryEscape(cfg.Key)
	r := requestJellyfin(t, h, root, "GET")
	if r.Code != http.StatusTemporaryRedirect {
		t.Fatalf("default must preserve working redirect, got %d", r.Code)
	}
}
func TestJellyfinDiagnosticsDoNotLeakKeysOrRemoteURLs(t *testing.T) {
	cfg := exampleCfg()
	cfg.StreamMode = "relay"
	h := newJellyfinHandler(cfg, func(string) ChannelData { return sampleFeed() })
	resp := requestJellyfin(t, h, "/jellyfin/diagnostics?key="+url.QueryEscape(cfg.Key), "GET")
	if resp.Code != 200 {
		t.Fatalf("diagnostics: %d", resp.Code)
	}
	var data map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if data["streamMode"] != "relay" {
		t.Fatalf("wrong mode %v", data["streamMode"])
	}
	if strings.Contains(resp.Body.String(), cfg.Key) || strings.Contains(resp.Body.String(), "https://streams.example.org/") {
		t.Fatal("diagnostics leaked key or upstream URL")
	}
	noAuth := requestJellyfin(t, h, "/jellyfin/diagnostics", "GET")
	if noAuth.Code != 401 {
		t.Fatal("diagnostics unauthenticated")
	}
}
func TestRelayPlaylistRejectsLocalAndProtectedContent(t *testing.T) {
	base, _ := url.Parse("https://streams.example.org/master.m3u8")
	cfg := exampleCfg()
	for _, body := range []string{
		"#EXTM3U\nhttp://127.0.0.1/private.ts\n",
		"#EXTM3U\n#EXT-X-KEY:METHOD=SAMPLE-AES,URI=\"https://provider.example/key\"\n",
		"#EXTM3U\n#EXT-X-MAP:URI=\"http://192.168.1.20/secret\"\n",
	} {
		if _, err := rewriteRelayPlaylist([]byte(body), base, cfg, "ch1"); err == nil {
			t.Fatalf("bad HLS playlist accepted: %q", body)
		}
	}
}

func TestJellyfinDirectModeDiagnosticsRecordSuccessfulRequests(t *testing.T) {
	cfg := exampleCfg()
	cfg.StreamMode = ""
	ch := sampleFeed().Channels[0]
	h := newJellyfinHandler(cfg, func(string) ChannelData { return ChannelData{Channels: []Channel{ch}} })
	path := "/jellyfin/stream/" + base64.RawURLEncoding.EncodeToString([]byte(ch.ID)) + ".m3u8?key=" + url.QueryEscape(cfg.Key)
	before := len(tuneEvents())
	head := requestJellyfin(t, h, path, "HEAD")
	if head.Code != 200 {
		t.Fatalf("HEAD code %d", head.Code)
	}
	get := requestJellyfin(t, h, path, "GET")
	if get.Code != http.StatusTemporaryRedirect {
		t.Fatalf("GET code %d", get.Code)
	}
	events := tuneEvents()
	if len(events) < before+2 || events[len(events)-2].Mode != "head" || events[len(events)-1].Mode != "redirect" {
		t.Fatalf("Direct-mode timing events missing: %#v", events)
	}
	response := requestJellyfin(t, h, "/jellyfin/diagnostics?key="+url.QueryEscape(cfg.Key), "GET")
	if response.Code != 200 {
		t.Fatalf("diagnostics: %d", response.Code)
	}
	var obj map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &obj); err != nil {
		t.Fatal(err)
	}
	if obj["streamMode"] != "redirect" {
		t.Fatalf("raw empty mode leaked: %#v", obj["streamMode"])
	}
	if obj["appVersion"] != version {
		t.Fatal("version not reported")
	}
	if _, ok := obj["hdhomerunUdpListening"].(bool); !ok {
		t.Fatalf("UDP bind status missing: %#v", obj)
	}
	if _, ok := obj["hdhomerunHttpListening"].(bool); !ok {
		t.Fatalf("HTTP bind status missing: %#v", obj)
	}
	if strings.Contains(response.Body.String(), cfg.Key) || strings.Contains(response.Body.String(), ch.Stream) {
		t.Fatal("diagnostics leaked key or stream URL")
	}
}

func TestJellyfinMediaRelayKeepsFFmpegHLSFileExtensions(t *testing.T) {
	cfg := exampleCfg()
	cfg.StreamMode = "mpegts"
	cfg.BaseURL = "http://127.0.0.1:45782"
	ch := sampleFeed().Channels[0]
	fixture := "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=600000\nhttps://streams.example.org/live/low.m3u8\n#EXT-X-MAP:URI=\"https://streams.example.org/live/init.mp4\"\n#EXTINF:4,\nhttps://streams.example.org/live/chunk.m4s?token=abc\n#EXTINF:4,\nhttps://streams.example.org/live/chunk-without-extension?token=def\n"
	base, _ := url.Parse("https://streams.example.org/live/master.m3u8")
	text, err := rewriteRelayPlaylist([]byte(fixture), base, cfg, ch.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"/resource.m3u8?", "/resource.mp4?", "/resource.m4s?", "/resource.ts?"} {
		if !strings.Contains(string(text), expected) {
			t.Fatalf("missing media extension %q: %s", expected, text)
		}
	}
	if strings.Contains(string(text), "token=abc") {
		t.Fatal("remote query should be URL-encoded in signed parameter")
	}
	// Every rewritten media URL still enforces the original channel-scoped HMAC.
	for _, raw := range strings.Split(string(text), "\n") {
		var utext string
		if strings.HasPrefix(raw, "http://127.0.0.1") {
			utext = raw
		} else if strings.Contains(raw, "URI=\"") {
			left := strings.Index(raw, "URI=\"") + len("URI=\"")
			right := strings.Index(raw[left:], "\"")
			if right >= 0 {
				utext = raw[left : left+right]
			}
		}
		if utext == "" {
			continue
		}
		u, er := url.Parse(utext)
		if er != nil {
			t.Fatal(er)
		}
		if !permittedStream(ch, u.Query().Get("u"), u.Query().Get("sig")) {
			t.Fatalf("invalid auth for %s", u.Path)
		}
	}
}

func TestRelaySendsCuratedProviderUserAgentToManifestsAndMedia(t *testing.T) {
	cfg := exampleCfg()
	cfg.StreamMode = "relay"
	cfg.BaseURL = "http://127.0.0.1:45782"
	ch := sampleFeed().Channels[0]
	ch.Stream = "https://streams.example.org/live/index.m3u8"
	ch.Headers = map[string]string{"user-agent": "otg/1.5.1 (AppleTv Apple TV 4; tvOS16.0; appletv.client)", "host": "evil.example.org"}
	fake := &relayFakeTransport{}
	old := mediaClient
	mediaClient = &http.Client{Transport: fake}
	t.Cleanup(func() { mediaClient = old })
	h := newJellyfinHandler(cfg, func(string) ChannelData { return ChannelData{Channels: []Channel{ch}} })
	id := base64.RawURLEncoding.EncodeToString([]byte(ch.ID))
	root := "/jellyfin/stream/" + id + ".m3u8?key=" + url.QueryEscape(cfg.Key)
	master := requestJellyfin(t, h, root, "GET")
	if master.Code != 200 {
		t.Fatalf("root %d", master.Code)
	}
	variant, err := url.Parse(strings.TrimSpace(strings.Split(master.Body.String(), "\n")[2]))
	if err != nil {
		t.Fatal(err)
	}
	nested := requestJellyfin(t, h, variant.RequestURI(), "GET")
	if nested.Code != 200 {
		t.Fatalf("nested %d", nested.Code)
	}
	var segment string
	for _, line := range strings.Split(nested.Body.String(), "\n") {
		if strings.HasPrefix(line, "http://127.") {
			segment = line
		}
	}
	su, err := url.Parse(segment)
	if err != nil {
		t.Fatal(err)
	}
	response := requestJellyfin(t, h, su.RequestURI(), "GET")
	if response.Code != 200 {
		t.Fatalf("segment %d", response.Code)
	}
	for _, ua := range fake.agents {
		if ua != ch.Headers["user-agent"] {
			t.Fatalf("wrong provider UA %q", ua)
		}
	}
}
func TestProviderHeadersIgnoreUntrustedValues(t *testing.T) {
	h := providerHeaders(map[string]string{"user-agent": "otg/1.5.1\r\nX-Evil: yes", "referer": "http://127.0.0.1/private", "cookie": "session=foo"})
	if len(h) != 0 {
		t.Fatalf("unsafe provider headers: %+v", h)
	}
}
