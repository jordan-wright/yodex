package paths

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOutPaths(t *testing.T) {
	base := t.TempDir()
	b := New(base)
	ts := time.Date(2025, 9, 30, 12, 0, 0, 0, time.UTC)

	wantDir := filepath.Join(base, "2025", "09", "30")
	if b.OutDir(ts) != wantDir {
		t.Fatalf("OutDir: got %q want %q", b.OutDir(ts), wantDir)
	}
	if b.EpisodeMarkdown(ts) != filepath.Join(wantDir, "episode.md") {
		t.Fatalf("EpisodeMarkdown path incorrect")
	}
	if b.EpisodeMP3(ts) != filepath.Join(wantDir, "episode.mp3") {
		t.Fatalf("EpisodeMP3 path incorrect")
	}
	if b.EpisodeMeta(ts) != filepath.Join(wantDir, "meta.json") {
		t.Fatalf("EpisodeMeta path incorrect")
	}
}

func TestEnsureOutDirAndOverwrite(t *testing.T) {
	base := t.TempDir()
	b := New(base)
	ts := time.Date(2025, 9, 30, 12, 0, 0, 0, time.UTC)

	// Ensure dir creates nested path
	if err := b.EnsureOutDir(ts); err != nil {
		t.Fatalf("EnsureOutDir error: %v", err)
	}
	// Create a file to simulate existing output
	md := b.EpisodeMarkdown(ts)
	if err := os.WriteFile(md, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write existing: %v", err)
	}
	// Check overwrite guard blocks when overwrite=false
	if err := CheckOverwrite([]string{md}, false); err == nil {
		t.Fatalf("expected overwrite guard to fail")
	}
	// When overwrite=true it should pass
	if err := CheckOverwrite([]string{md}, true); err != nil {
		t.Fatalf("overwrite=true should not error: %v", err)
	}
}
