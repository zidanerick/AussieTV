package teletext

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testConfig() Config {
	return Config{Service: "AUSSIE TEXT", Region: "Melbourne", IndexRefreshSeconds: 30, DataRefreshSeconds: 300, Sections: []Section{{Start: 101, End: 103, Title: "NOTICES", Kind: "notices", Live: true}, {Start: 120, End: 139, Title: "WEATHER", Kind: "weather", Live: true}, {Start: 180, End: 199, Title: "GUIDE", Kind: "schedule", Live: true}, {Start: 200, End: 200, Title: "TEST", Kind: "diagnostic", Live: true}, {Start: 800, End: 801, Title: "CC", Kind: "captions"}, {Start: 900, End: 999, Title: "ADMIN", Kind: "admin"}}, Pages: []PageDef{{Number: 101, Title: "NOTICE", Lines: []string{"TEST SERVICE"}}}}
}
func TestPagesAre24x40ASCIIAndPrivatePagesBlocked(t *testing.T) {
	s := New(testConfig(), nil)
	for _, n := range []int{100, 101, 120, 180, 200, 800} {
		p, e := s.Page(n)
		if e != nil {
			t.Fatal(n, e)
		}
		if len(p.Rows) != 24 {
			t.Fatal("row count")
		}
		for i, line := range p.Rows {
			if len(line.Text) != 40 {
				t.Fatalf("page %d row %d is %d columns", n, i, len(line.Text))
			}
			for _, b := range []byte(line.Text) {
				if b < 32 || b > 126 {
					t.Fatalf("non-ascii printable page %d: %x", n, b)
				}
			}
		}
		if !strings.Contains(p.Rows[23].Text, "UPD") {
			t.Fatal("missing timestamp")
		}
	}
	if _, e := s.Page(900); e == nil {
		t.Fatal("public admin page leak")
	}
}
func TestTTIFormatAndEncoding(t *testing.T) {
	s := New(testConfig(), nil)
	p, _ := s.Page(100)
	b := TTI(p)
	if !bytes.HasPrefix(b, []byte("PN,10000\r\nPS,8000\r\nCT,1,C\r\n")) {
		t.Fatal("bad TTI header")
	}
	if bytes.Count(b, []byte("OL,")) != 24 {
		t.Fatalf("missing TTI rows: %s", b)
	}
	for _, line := range bytes.Split(bytes.TrimSpace(b), []byte("\r\n")) {
		if !bytes.HasPrefix(line, []byte("OL,")) {
			continue
		}
		data := line[bytes.IndexByte(line, ',')+1:]
		data = data[bytes.IndexByte(data, ',')+1:]
		if len(data) != 41 || data[0] != 27 {
			t.Fatalf("bad output line: %q %d", data, len(data))
		}
		for _, b := range data {
			if b > 127 {
				t.Fatalf("8-bit TTI control: %x", b)
			}
		}
	}
}
func TestIndexListingsAndNoAdminExport(t *testing.T) {
	s := New(testConfig(), nil)
	p, _ := s.Page(100)
	joined := ""
	for _, r := range p.Rows {
		joined += r.Text
	}
	if !strings.Contains(joined, "120 WEATHER") || !strings.Contains(joined, "180 GUIDE") {
		t.Fatal("missing live magazine", joined)
	}
	b, e := s.ExportZip()
	if e != nil || len(b) == 0 {
		t.Fatal(e)
	}
	if bytes.Contains(b, []byte("900.tti")) {
		t.Fatal("admin leaked")
	}
}
func TestHTTPPublicPagesAndETag(t *testing.T) {
	s := New(testConfig(), nil)
	h := s.Handler()
	r := httptest.NewRequest("GET", "/teletext/page/100", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	var p Page
	if e := json.Unmarshal(w.Body.Bytes(), &p); e != nil {
		t.Fatal(e)
	}
	r = httptest.NewRequest("GET", "/teletext/page/100", nil)
	r.Header.Set("If-None-Match", w.Header().Get("ETag"))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNotModified {
		t.Fatal("ETag invalidation", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/teletext/page/900", nil))
	if w.Code != 403 {
		t.Fatal("admin not blocked")
	}
}
func TestPageCacheTime(t *testing.T) {
	s := New(testConfig(), nil)
	now := time.Date(2026, 11, 18, 12, 0, 5, 0, time.UTC)
	a, _ := s.render(200, now, false)
	b, _ := s.render(200, now.Add(2*time.Second), false)
	if a.Revision != b.Revision {
		t.Fatal("data page revision changed between refresh intervals")
	}
}
