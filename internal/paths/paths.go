package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultBaseDir         = "out"
	defaultEpisodeFilename = "episode.md"
	defaultMP3Filename     = "episode.mp3"
	defaultMetaFilename    = "meta.json"
)

// Builder constructs output paths rooted at Base (default "out").
type Builder struct {
	Base string
}

func New(base string) *Builder {
	if base == "" {
		base = defaultBaseDir
	}
	return &Builder{Base: base}
}

// OutDir returns the date-based output directory: Base/YYYY/MM/DD
func (b *Builder) OutDir(t time.Time) string {
	y, m, d := t.UTC().Date()
	return filepath.Join(b.Base, fmt.Sprintf("%04d", y), fmt.Sprintf("%02d", int(m)), fmt.Sprintf("%02d", d))
}

func (b *Builder) EpisodeMarkdown(t time.Time) string {
	return filepath.Join(b.OutDir(t), defaultEpisodeFilename)
}
func (b *Builder) EpisodeMP3(t time.Time) string {
	return filepath.Join(b.OutDir(t), defaultMP3Filename)
}
func (b *Builder) EpisodeMeta(t time.Time) string {
	return filepath.Join(b.OutDir(t), defaultMetaFilename)
}

// EnsureOutDir creates the date-based directory if it does not exist.
func (b *Builder) EnsureOutDir(t time.Time) error {
	dir := b.OutDir(t)
	return os.MkdirAll(dir, 0o755)
}

// CheckOverwrite enforces overwrite behavior. If any path exists and overwrite is false, returns error.
func CheckOverwrite(paths []string, overwrite bool) error {
	if overwrite {
		return nil
	}
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return fmt.Errorf("refusing to overwrite existing file: %s (use --overwrite)", p)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("checking file: %s: %w", p, err)
		}
	}
	return nil
}
