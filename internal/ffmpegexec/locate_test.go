package ffmpegexec

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindNearPrioritizesBundledBinary(t *testing.T) {
	root := t.TempDir()
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}
	bundled := filepath.Join(root, "ffmpeg", name)
	if err := os.MkdirAll(filepath.Dir(bundled), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bundled, []byte("test"), 0700); err != nil {
		t.Fatal(err)
	}
	if got := findNear(root); got != bundled {
		t.Fatalf("want bundled %q, got %q", bundled, got)
	}
	if err := os.Remove(bundled); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		direct := filepath.Join(root, name)
		if err := os.WriteFile(direct, []byte("test"), 0700); err != nil {
			t.Fatal(err)
		}
		if got := findNear(root); got != direct {
			t.Fatalf("want adjacent %q, got %q", direct, got)
		}
	}

}
func TestOverrideRejectsRelativeAndMissing(t *testing.T) {
	t.Setenv("AUSSIETV_FFMPEG_PATH", "relative/ffmpeg.exe")
	if _, err := Find(); err == nil {
		t.Fatal("relative path was accepted")
	}
	t.Setenv("AUSSIETV_FFMPEG_PATH", filepath.Join(t.TempDir(), "missing.exe"))
	if _, err := Find(); err == nil {
		t.Fatal("missing override was accepted")
	}
}

func TestFindConfiguredFFmpegLocation(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "ffmpeg-custom.exe")
	if err := os.WriteFile(exe, []byte("fixture"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ffmpeg-location.txt"), []byte(exe+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := findConfigured(dir); got != exe {
		t.Fatalf("got %q want %q", got, exe)
	}
	if err := os.WriteFile(filepath.Join(dir, "ffmpeg-location.txt"), []byte("relative/ffmpeg.exe"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := findConfigured(dir); got != "" {
		t.Fatalf("relative path accepted: %q", got)
	}
}
