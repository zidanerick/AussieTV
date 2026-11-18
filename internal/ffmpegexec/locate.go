// Package ffmpegexec locates a self-contained FFmpeg for portable AussieTV.
// The adjacent portable binary takes precedence over an unrelated system PATH.
package ffmpegexec

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Find returns an FFmpeg executable without requiring PATH modifications.
// AUSSIETV_FFMPEG_PATH is an optional explicit absolute path for administrators.
func Find() (string, error) {
	if path := os.Getenv("AUSSIETV_FFMPEG_PATH"); path != "" {
		if !filepath.IsAbs(path) {
			return "", errors.New("AUSSIETV_FFMPEG_PATH must be absolute")
		}
		if available(path) {
			return path, nil
		}
		return "", errors.New("AUSSIETV_FFMPEG_PATH points to a missing/non-executable file")
	}
	exe, err := os.Executable()
	if err == nil {
		if path := findNear(filepath.Dir(exe)); path != "" {
			return path, nil
		}
		// The installed wizard may place FFmpeg outside the application folder.
		// Its verified downloader records the chosen absolute executable path.
		if path := findConfigured(filepath.Dir(exe)); path != "" {
			return path, nil
		}
	}
	return exec.LookPath("ffmpeg")
}

func findNear(dir string) string {
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}
	for _, candidate := range []string{filepath.Join(dir, "ffmpeg", name), filepath.Join(dir, name)} {
		if available(candidate) {
			return candidate
		}
	}
	return ""
}

func available(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	return runtime.GOOS == "windows" || info.Mode().Perm()&0111 != 0
}

// findConfigured reads the local installer-generated path hint, not a URL or
// command line. The installation directory must be under the user's control.
func findConfigured(appDir string) string {
	raw, err := os.ReadFile(filepath.Join(appDir, "ffmpeg-location.txt"))
	if err != nil || len(raw) > 4096 {
		return ""
	}
	path := strings.TrimSpace(string(raw))
	if path == "" || strings.ContainsAny(path, "\x00\r\n") || !filepath.IsAbs(path) {
		return ""
	}
	if available(path) {
		return path
	}
	return ""
}
