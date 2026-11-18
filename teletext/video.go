package teletext

import (
	"aussietv/internal/ffmpegexec"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// HasVideoEncoder is deliberately opt-in; it supports a bundled executable.
func HasVideoEncoder() bool {
	if os.Getenv("AUSSIETV_TELETEXT_VIDEO") != "1" {
		return false
	}
	_, err := ffmpegexec.Find()
	return err == nil
}

// ServeMPEGTS converts the visible Teletext pages into a PAL-resolution,
// 25-fps *in-vision* channel, playable through Jellyfin's M3U tuner. This is
// NOT an embedded DVB-TXT elementary stream or actual composite VBI signal.
func (s *Service) ServeMPEGTS(w http.ResponseWriter, r *http.Request) error {
	if !HasVideoEncoder() {
		http.Error(w, "Teletext video requires FFmpeg and AUSSIETV_TELETEXT_VIDEO=1", http.StatusServiceUnavailable)
		return errors.New("video encoder not enabled")
	}
	dir, err := os.MkdirTemp("", "aussietv-teletext-video-")
	if err != nil {
		http.Error(w, "Temporary renderer unavailable", 500)
		return err
	}
	defer os.RemoveAll(dir)
	txt := filepath.Join(dir, "page.txt")
	pages := []int{100, 120, 180, 200, 300, 400}
	next := 0
	update := func() {
		for i := 0; i < len(pages); i++ {
			id := pages[(next+i)%len(pages)]
			p, e := s.Page(id)
			if e != nil {
				continue
			}
			var b strings.Builder
			for n, l := range p.Rows {
				if n > 0 {
					b.WriteString("\n")
				}
				b.WriteString(l.Text)
			}
			temp := txt + ".tmp"
			if e = os.WriteFile(temp, []byte(b.String()), 0600); e == nil {
				_ = os.Rename(temp, txt)
			}
			next = (next + i + 1) % len(pages)
			return
		}
	}
	update()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go func() {
		tick := time.NewTicker(12 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				update()
			}
		}
	}()
	fontFile := "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"
	if runtime.GOOS == "windows" {
		fontFile = "C\\:/Windows/Fonts/consola.ttf"
	}
	if supplied := os.Getenv("AUSSIETV_TELETEXT_FONT"); supplied != "" {
		fontFile = strings.ReplaceAll(supplied, "\\", "/")
		fontFile = strings.ReplaceAll(fontFile, ":", "\\:")
	}
	filter := fmt.Sprintf("drawtext=fontfile=%s:textfile=page.txt:reload=1:fontcolor=white:fontsize=21:x=22:y=18:line_spacing=1", fontFile)
	args := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-re", "-f", "lavfi", "-i", "color=c=0x080b19:s=720x576:r=25", "-f", "lavfi", "-i", "anullsrc=r=48000:cl=stereo", "-vf", filter, "-c:v", "mpeg2video", "-pix_fmt", "yuv420p", "-q:v", "7", "-g", "25", "-c:a", "mp2", "-b:a", "64k", "-f", "mpegts", "pipe:1"}
	binary, err := ffmpegexec.Find()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	ffmpegexec.Configure(cmd)
	cmd.Dir = dir
	cmd.Stdout = w
	// No API credentials are supplied on FFmpeg command lines or in page text.
	cmd.Stderr = os.Stderr
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	return cmd.Run()
}
