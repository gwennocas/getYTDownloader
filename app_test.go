package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadArgsPlaylistAndSingle(t *testing.T) {
	folder := filepath.Join("C:", "Downloads")
	base := DownloadOptions{URL: "https://example.com/watch?v=123&list=abc", Mode: "video", Quality: "720"}
	single := strings.Join(buildDownloadArgs(folder, base, ""), "\n")
	if !strings.Contains(single, "--no-playlist") || strings.Contains(single, "--download-archive") {
		t.Fatalf("single download flags: %s", single)
	}
	if !strings.Contains(single, "best[height<=720][ext=mp4]") {
		t.Fatalf("missing non-FFmpeg fallback: %s", single)
	}
	base.Playlist = true
	playlist := strings.Join(buildDownloadArgs(folder, base, filepath.Join(folder, "ffmpeg.exe")), "\n")
	if !strings.Contains(playlist, "--yes-playlist") || !strings.Contains(playlist, "--download-archive") || !strings.Contains(playlist, "--merge-output-format") {
		t.Fatalf("playlist flags: %s", playlist)
	}
}

func TestVersionComparison(t *testing.T) {
	for _, tc := range []struct {
		latest, current string
		want            bool
	}{
		{"v1.2.0", "v1.1.9", true},
		{"v1.2.0", "v1.2.0", false},
		{"v1.1.9", "v1.2.0", false},
		{"v1.0.0", "dev", true},
	} {
		if got := newerVersion(tc.latest, tc.current); got != tc.want {
			t.Errorf("newerVersion(%q, %q) = %v, want %v", tc.latest, tc.current, got, tc.want)
		}
	}
}

func TestURLValidation(t *testing.T) {
	for _, raw := range []string{"file:///C:/secret", "javascript:alert(1)", "https://", "https://user:pass@example.com"} {
		if validateURL(raw) == nil {
			t.Errorf("accepted invalid URL %q", raw)
		}
	}
	if err := validateURL("https://www.youtube.com/playlist?list=123"); err != nil {
		t.Fatal(err)
	}
}
