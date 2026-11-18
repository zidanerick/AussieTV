package teletext

import (
	"bytes"
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestTeletextMPEGTSOptIn(t *testing.T) {
	if _, e := exec.LookPath("ffmpeg"); e != nil {
		t.Skip("ffmpeg unavailable")
	}
	t.Setenv("AUSSIETV_TELETEXT_VIDEO", "1")
	if !HasVideoEncoder() {
		t.Fatal("ffmpeg missing")
	}
	c := testConfig()
	s := New(c, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	r := httptest.NewRequest("GET", "/jellyfin/teletext.ts", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	_ = s.ServeMPEGTS(w, r)
	b := w.Body.Bytes()
	if len(b) < 188 || b[0] != 0x47 {
		t.Fatalf("MPEG TS unavailable: %d bytes, prefix %q", len(b), b[:min(len(b), 60)])
	}
	if !bytes.Contains(b, []byte{0x47}) {
		t.Fatal("invalid stream")
	}
	_ = os.ErrNotExist
}
