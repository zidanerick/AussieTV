// Package teletext models a 40-column, 24-row information service and emits
// VBIT2 TTI page files. It does not modulate PAL composite/VBI signals.
package teletext

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const Width = 40
const Height = 24

type Section struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
	Live  bool   `json:"live"`
}
type PageDef struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Lines  []string `json:"lines"`
}
type WeatherDef struct {
	Page           int    `json:"page"`
	City           string `json:"city"`
	ObservationURL string `json:"observation_url"`
}
type Config struct {
	Service             string       `json:"service"`
	Region              string       `json:"region"`
	IndexRefreshSeconds int          `json:"index_refresh_seconds"`
	DataRefreshSeconds  int          `json:"data_refresh_seconds"`
	Sections            []Section    `json:"sections"`
	Pages               []PageDef    `json:"pages"`
	Weather             []WeatherDef `json:"weather"`
}
type Programme struct {
	Name       string
	Start, End time.Time
}
type Channel struct {
	Name, Network, Number string
	Programmes            []Programme
}
type GuideFunc func(region string) []Channel

// Line stores exactly forty 7-bit display characters. Colour is metadata;
// TTI exporter writes Teletext control bytes in place of a display column.
type Line struct {
	Text   string `json:"text"`
	Colour string `json:"colour"`
}
type Page struct {
	Number   int          `json:"number"`
	Title    string       `json:"title"`
	Rows     [Height]Line `json:"rows"`
	Updated  time.Time    `json:"updated"`
	Revision string       `json:"revision"`
}
type fetched struct {
	lines []string
	at    time.Time
	err   error
}
type Service struct {
	cfg     Config
	guide   GuideFunc
	client  *http.Client
	mu      sync.Mutex
	weather map[int]fetched
}

func LoadConfig(data []byte) (Config, error) {
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	if strings.TrimSpace(c.Service) == "" {
		return c, errors.New("service name required")
	}
	if c.IndexRefreshSeconds < 5 {
		c.IndexRefreshSeconds = 30
	}
	if c.DataRefreshSeconds < 60 {
		c.DataRefreshSeconds = 300
	}
	for _, s := range c.Sections {
		if s.Start < 100 || s.End > 999 || s.Start > s.End {
			return c, fmt.Errorf("invalid section %d-%d", s.Start, s.End)
		}
	}
	for _, p := range c.Pages {
		if p.Number < 100 || p.Number > 999 {
			return c, fmt.Errorf("invalid page %d", p.Number)
		}
	}
	for _, w := range c.Weather {
		if w.Page < 120 || w.Page > 139 {
			return c, fmt.Errorf("invalid weather page %d", w.Page)
		}
	}
	return c, nil
}
func New(c Config, guide GuideFunc) *Service {
	return &Service{cfg: c, guide: guide, weather: make(map[int]fetched), client: &http.Client{Timeout: 7 * time.Second}}
}
func (s *Service) Config() Config  { return s.cfg }
func row(text, colour string) Line { return Line{Text: ascii40(text), Colour: colour} }
func ascii40(text string) string {
	var b strings.Builder
	for _, r := range text {
		if b.Len() >= Width {
			break
		}
		if r < 32 || r > 126 {
			if r == '\t' {
				b.WriteByte(' ')
			} else {
				b.WriteByte('?')
			}
		} else {
			b.WriteByte(byte(r))
		}
	}
	for b.Len() < Width {
		b.WriteByte(' ')
	}
	return b.String()
}
func (s *Service) Section(number int) (Section, bool) {
	if number == 100 {
		return Section{Start: 100, End: 100, Title: "MAIN INDEX", Kind: "index", Live: true}, true
	}
	for _, t := range s.cfg.Sections {
		if number >= t.Start && number <= t.End {
			return t, true
		}
	}
	return Section{}, false
}
func (s *Service) Listed() []int {
	result := []int{100}
	set := map[int]bool{100: true}
	for _, t := range s.cfg.Sections {
		if t.Live && t.Start < 900 && !set[t.Start] {
			result = append(result, t.Start)
			set[t.Start] = true
		}
	}
	for _, t := range s.cfg.Pages {
		if t.Number < 900 && !set[t.Number] {
			result = append(result, t.Number)
			set[t.Number] = true
		}
	}
	for _, t := range s.cfg.Weather {
		if t.Page < 900 && !set[t.Page] {
			result = append(result, t.Page)
			set[t.Page] = true
		}
	}
	sort.Ints(result)
	return result
}
func (s *Service) render(number int, now time.Time, admin bool) (Page, error) {
	sec, ok := s.Section(number)
	if !ok {
		return Page{}, os.ErrNotExist
	}
	if number >= 900 && !admin {
		return Page{}, os.ErrPermission
	}
	refresh := s.cfg.DataRefreshSeconds
	if number == 100 {
		refresh = s.cfg.IndexRefreshSeconds
	}
	stamp := now.Truncate(time.Duration(refresh) * time.Second)
	page := Page{Number: number, Title: sec.Title, Updated: stamp}
	page.Rows[0] = row(fmt.Sprintf("%03d %s", number, strings.ToUpper(sec.Title)), "yellow")
	page.Rows[1] = row("", "white")
	set := func(i int, text, colour string) {
		if i >= 2 && i <= 22 {
			page.Rows[i] = row(text, colour)
		}
	}
	for i := 2; i <= 22; i++ {
		page.Rows[i] = row("", "white")
	}
	for _, p := range s.cfg.Pages {
		if p.Number == number {
			page.Title = p.Title
			page.Rows[0] = row(fmt.Sprintf("%03d %s", number, strings.ToUpper(p.Title)), "yellow")
			for i, text := range p.Lines {
				if i < 21 {
					set(i+2, text, "white")
				}
			}
			break
		}
	}
	switch sec.Kind {
	case "index":
		set(2, "WELCOME TO "+strings.ToUpper(s.cfg.Service), "yellow")
		set(3, now.In(timezone()).Format("MON 02 JAN 15:04 MST"), "cyan")
		set(5, "MAGAZINES", "yellow")
		y := 6
		for _, block := range s.cfg.Sections {
			if !block.Live || block.Start >= 900 || block.Start == 200 {
				continue
			}
			if y >= 21 {
				break
			}
			set(y, fmt.Sprintf("%03d %s", block.Start, block.Title), "cyan")
			y++
		}
		set(21, "200 TEST AND DIAGNOSTICS", "cyan")
	case "diagnostic":
		set(2, "40 COLUMN / 24 ROW SERVICE OK", "yellow")
		set(4, "OUTPUT: ASCII / TTI (VBIT2)", "white")
		set(5, "REGION: "+s.cfg.Region, "white")
		set(6, "DATA REFRESH: "+strconv.Itoa(s.cfg.DataRefreshSeconds)+" SECONDS", "white")
		set(8, "PAL VBI: EXTERNAL ENCODER REQUIRED", "cyan")
		set(9, "JELLYFIN: VIEWER OR VIDEO ENCODER", "cyan")
	case "weather":
		matched := false
		for _, w := range s.cfg.Weather {
			if w.Page == number {
				matched = true
				page.Title = "BOM WEATHER " + strings.ToUpper(w.City)
				page.Rows[0] = row(fmt.Sprintf("%03d BOM %s", number, strings.ToUpper(w.City)), "yellow")
				d := s.weatherData(w)
				for i, line := range d.lines {
					if i < 18 {
						set(i+2, line, "white")
					}
				}
				if len(d.lines) == 0 {
					set(2, "BOM FEED UNAVAILABLE", "yellow")
				}
				if !d.at.IsZero() {
					page.Updated = d.at
				}
				break
			}
		}
		if !matched {
			set(2, "BOM CITY NOT CONFIGURED", "yellow")
		}
	case "schedule", "schedule_sbs", "schedule_seven":
		if s.guide == nil {
			set(2, "EPG SOURCE NOT CONFIGURED", "yellow")
			break
		}
		chans := s.guide(s.cfg.Region)
		var group []Channel
		for _, ch := range chans {
			if sec.Kind == "schedule_sbs" && ch.Network != "SBS" {
				continue
			}
			if sec.Kind == "schedule_seven" && ch.Network != "Seven" {
				continue
			}
			group = append(group, ch)
		}
		if len(group) == 0 {
			set(2, "EPG DATA UNAVAILABLE", "yellow")
			break
		}
		idx := number - sec.Start
		if idx >= len(group) {
			set(2, "NO CHANNEL ASSIGNED TO THIS PAGE", "yellow")
			break
		}
		ch := group[idx]
		set(2, fmt.Sprintf("%s  CH %s", strings.ToUpper(ch.Name), ch.Number), "yellow")
		n := 4
		for _, p := range ch.Programmes {
			if p.End.Before(now) {
				continue
			}
			if n > 20 {
				break
			}
			set(n, p.Start.In(timezone()).Format("MON 15:04")+" "+p.Name, "white")
			n++
		}
		if n == 4 {
			set(4, "PROGRAMME DATA UNAVAILABLE", "yellow")
		}
	case "news", "results", "general":
		if len(s.lookupLines(number)) == 0 {
			set(2, "CONTENT SOURCE UNAVAILABLE", "yellow")
			set(4, "EDIT config/teletext.json", "cyan")
		}
	case "captions":
		set(2, "NO SYNCHRONISED CAPTION INGEST", "yellow")
		set(4, "PAGES 800-801 RESERVED FOR CC", "white")
	case "reserved":
		set(2, "PAGE RESERVED FOR EXPANSION", "yellow")
	case "admin":
		set(2, "PRIVATE ADMINISTRATIVE PAGE", "yellow")
	}
	for i, line := range s.lookupLines(number) {
		if i < 18 {
			set(i+4, line, "white")
		}
	}
	updated := page.Updated.In(timezone()).Format("02JAN 15:04")
	footer := "100 INDEX   " + fmt.Sprintf("%03d", number) + "          UPD " + updated
	page.Rows[23] = row(footer, "cyan")
	// Revision is a semantic hash of visible bytes, ignoring time-of-request.
	hash := sha256.New()
	for _, l := range page.Rows {
		hash.Write([]byte(l.Text + l.Colour))
	}
	page.Revision = fmt.Sprintf("%x", hash.Sum(nil))[:16]
	return page, nil
}
func (s *Service) lookupLines(number int) []string {
	for _, p := range s.cfg.Pages {
		if p.Number == number {
			return p.Lines
		}
	}
	return nil
}
func timezone() *time.Location {
	l, e := time.LoadLocation("Australia/Melbourne")
	if e != nil {
		return time.FixedZone("AEST", 10*3600)
	}
	return l
}
func (s *Service) Page(number int) (Page, error) { return s.render(number, time.Now(), false) }

// Weather JSON is from an official BOM observation product. It must never be
// silently replaced with invented readings or used when its source is missing.
func (s *Service) weatherData(cfg WeatherDef) fetched {
	if cfg.ObservationURL == "" {
		return fetched{lines: []string{"BOM FEED NOT CONFIGURED"}}
	}
	s.mu.Lock()
	old, ok := s.weather[cfg.Page]
	if ok && time.Since(old.at) < time.Duration(s.cfg.DataRefreshSeconds)*time.Second {
		s.mu.Unlock()
		return old
	}
	s.mu.Unlock()
	result := fetched{at: time.Now(), lines: []string{"BOM OBSERVATIONS UNAVAILABLE"}}
	u, err := url.Parse(cfg.ObservationURL)
	if err != nil || u.Scheme != "https" || (u.Hostname() != "bom.gov.au" && !strings.HasSuffix(u.Hostname(), ".bom.gov.au")) || u.User != nil || u.Port() != "" {
		result.err = errors.New("weather URL must use the official BOM HTTPS domain")
		return result
	}
	// DNS should never permit private, loopback or link-local weather targets.
	c := *s.client
	c.Transport = &http.Transport{Proxy: nil, TLSHandshakeTimeout: 5 * time.Second, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(address)
		if e != nil {
			return nil, e
		}
		ips, e := net.DefaultResolver.LookupIPAddr(ctx, host)
		if e != nil {
			return nil, e
		}
		var d net.Dialer
		for _, a := range ips {
			ip := a.IP
			if ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
				return d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			}
		}
		return nil, errors.New("blocked weather endpoint address")
	}}
	c.CheckRedirect = func(r *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return errors.New("too many weather redirects")
		}
		v := r.URL
		if v.Scheme != "https" || (v.Hostname() != "bom.gov.au" && !strings.HasSuffix(v.Hostname(), ".bom.gov.au")) {
			return errors.New("weather redirect left BOM domain")
		}
		return nil
	}
	r, e := c.Get(cfg.ObservationURL)
	if e != nil {
		result.err = e
	} else {
		defer r.Body.Close()
		if r.StatusCode == 200 {
			var data struct {
				Observations struct {
					Data []struct {
						LocalDateTime string `json:"local_date_time"`
						AirTemp       any    `json:"air_temp"`
						RelHum        any    `json:"rel_hum"`
						WindDir       string `json:"wind_dir"`
						WindSpd       any    `json:"wind_spd_kmh"`
						Gust          any    `json:"gust_kmh"`
						Rain          any    `json:"rain_trace"`
					} `json:"data"`
				} `json:"observations"`
			}
			if e = json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(&data); e == nil && len(data.Observations.Data) > 0 {
				o := data.Observations.Data[0]
				result.lines = []string{
					"BOM OBSERVATIONS - " + strings.ToUpper(cfg.City),
					"OBSERVED: " + o.LocalDateTime,
					"TEMP: " + fmt.Sprint(o.AirTemp) + " C",
					"HUMIDITY: " + fmt.Sprint(o.RelHum) + " %",
					"WIND: " + o.WindDir + " " + fmt.Sprint(o.WindSpd) + " KM/H",
					"GUST: " + fmt.Sprint(o.Gust) + " KM/H",
					"RAIN SINCE 9AM: " + fmt.Sprint(o.Rain) + " MM",
					"5-DAY FORECAST: FEED NOT CONFIGURED",
					"SUNRISE/SUNSET: NOT CONFIGURED",
					"SOURCE: BUREAU OF METEOROLOGY",
				}
			} else {
				result.err = errors.New("BOM JSON could not be decoded")
			}
		} else {
			result.err = fmt.Errorf("BOM HTTP %d", r.StatusCode)
		}
	}
	s.mu.Lock()
	s.weather[cfg.Page] = result
	s.mu.Unlock()
	return result
}

// TTI conforms to the MRG Systems text line format understood by VBIT2.
// Control bytes are encoded as 7-bit ESC plus colour code, occupying one cell.
func TTI(p Page) []byte {
	var out bytes.Buffer
	fmt.Fprintf(&out, "PN,%03d00\r\nPS,8000\r\nCT,1,C\r\n", p.Number)
	for i, l := range p.Rows {
		code := byte('G') // white text (0x07)
		if l.Colour == "yellow" {
			code = 'C'
		} // 0x03
		if l.Colour == "cyan" {
			code = 'F'
		} // 0x06
		fmt.Fprintf(&out, "OL,%d,%c%c%s\r\n", i+1, 27, code, l.Text[:39])
	}
	if p.Number == 100 {
		out.WriteString("FL,100,120,180,200,300,400\r\n")
	}
	return out.Bytes()
}
func (s *Service) ExportZip() ([]byte, error) {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, n := range s.Listed() {
		if n >= 800 {
			continue
		} // No fake live captions or internal pages.
		p, e := s.Page(n)
		if e != nil {
			return nil, e
		}
		f, e := z.Create(fmt.Sprintf("%03d.tti", n))
		if e != nil {
			return nil, e
		}
		if _, e = f.Write(TTI(p)); e != nil {
			return nil, e
		}
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /teletext/page/{number}", func(w http.ResponseWriter, r *http.Request) {
		n, e := strconv.Atoi(r.PathValue("number"))
		if e != nil || n < 100 || n > 999 {
			http.NotFound(w, r)
			return
		}
		p, e := s.Page(n)
		if errors.Is(e, os.ErrPermission) {
			http.Error(w, "Administrative page", 403)
			return
		}
		if e != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("ETag", `"`+p.Revision+`"`)
		if r.Header.Get("If-None-Match") == `"`+p.Revision+`"` {
			w.WriteHeader(304)
			return
		}
		_ = json.NewEncoder(w).Encode(p)
	})
	mux.HandleFunc("GET /teletext/tti/{number}", func(w http.ResponseWriter, r *http.Request) {
		n, e := strconv.Atoi(r.PathValue("number"))
		if e != nil {
			http.NotFound(w, r)
			return
		}
		p, e := s.Page(n)
		if errors.Is(e, os.ErrPermission) {
			http.Error(w, "Administrative page", 403)
			return
		}
		if e != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=us-ascii")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%03d.tti\"", n))
		w.Write(TTI(p))
	})
	mux.HandleFunc("GET /teletext/export.zip", func(w http.ResponseWriter, r *http.Request) {
		z, e := s.ExportZip()
		if e != nil {
			http.Error(w, "TTI export unavailable", 500)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=\"aussietv-teletext-pages.zip\"")
		w.Write(z)
	})
	mux.HandleFunc("GET /teletext/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Intentionally self-contained: usable without JavaScript but refreshed by it.
		io.WriteString(w, viewerHTML)
	})
	return mux
}

const viewerHTML = `<!doctype html><html lang="en-AU"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>AussieTV Teletext</title><style>body{background:#080b19;color:#eee;font-family:monospace;text-align:center;margin:10px}h1{font-size:16px;color:#ffda42}#screen{display:inline-block;background:#000;padding:12px;text-align:left;max-width:100%;overflow:auto}pre{font:clamp(9px,2vw,20px)/1.25 monospace;white-space:pre;margin:0}.yellow{color:#ffda48}.cyan{color:#76eaff}.white{color:white}input,button{font:18px monospace;background:#101b34;color:white;border:1px solid #7684a7;padding:8px;margin:3px}button:focus-visible{outline:3px solid yellow}</style></head><body><h1>AUSSIE TEXT - 40x24</h1><label>PAGE <input id="page" size="3" value="100" maxlength="3" inputmode="numeric"></label><button id="go">GO</button><button id="prev">&lt;</button><button id="next">&gt;</button><a href="/teletext/export.zip" id="zip"><button>TTI ZIP</button></a><p>Keys: 3-digit page + Enter, arrows to change pages</p><a href="/" style="color:#76eaff">BACK TO TV</a><div id="screen" role="region" aria-label="Teletext page"><pre id="rows">Loading...</pre></div><script>let p=100;const qs=location.search;document.getElementById('zip').href='/teletext/export.zip'+qs;async function load(n){if(n<100||n>899)return;let r=await fetch('/teletext/page/'+n+qs,{cache:'no-store'});if(!r.ok)return;let d=await r.json();p=n;document.getElementById('page').value=n;document.getElementById('rows').innerHTML=d.rows.map(x=>'<span class="'+x.colour+'">'+x.text.replace(/[&<>]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;'}[c]))+'</span>').join('\n');}document.getElementById('go').onclick=()=>load(Number(document.getElementById('page').value));document.getElementById('prev').onclick=()=>load(p-1);document.getElementById('next').onclick=()=>load(p+1);document.addEventListener('keydown',e=>{if(e.key==='Escape')location.href='/';if(e.key==='Enter')load(Number(document.getElementById('page').value));if(e.key==='ArrowLeft'&&e.target.tagName!=='INPUT')load(p-1);if(e.key==='ArrowRight'&&e.target.tagName!=='INPUT')load(p+1)});load(100);setInterval(()=>load(p),30000);</script></body></html>`

// Escape page contents for any downstream HTML clients not using textContent.
func Escape(s string) string { return html.EscapeString(s) }
