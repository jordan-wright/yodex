package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"yodex/internal/ai"
	cfgpkg "yodex/internal/config"
	"yodex/internal/paths"
)

type fakeTTSClient struct {
	lastModel string
	lastVoice string
	lastText  string
	calls     int
}

func (f *fakeTTSClient) TTS(ctx context.Context, model, voice, text string, w io.Writer) error {
	f.lastModel = model
	f.lastVoice = voice
	f.lastText = text
	f.calls++
	_, err := w.Write([]byte("mp3bytes"))
	return err
}

func TestAudioWritesMP3(t *testing.T) {
	origPausePath := pauseAudioPath
	t.Cleanup(func() { pauseAudioPath = origPausePath })
	pauseAudioPath = filepath.Join(t.TempDir(), "pause10s.mp3")
	if err := os.WriteFile(pauseAudioPath, []byte("pausebytes"), 0o644); err != nil {
		t.Fatalf("write pause audio: %v", err)
	}
	origClient := newTTSClient
	t.Cleanup(func() { newTTSClient = origClient })

	fake := &fakeTTSClient{}
	newTTSClient = func(cfg cfgpkg.Config) (ai.TTSClient, error) {
		return fake, nil
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })

	date := time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC)
	builder := paths.New("")
	if err := builder.EnsureOutDir(date); err != nil {
		t.Fatalf("EnsureOutDir: %v", err)
	}
	sectionData := map[string]string{
		"intro": "Hello there.",
		"topic": "Topic time. [long pause] Nice idea.",
		"game":  "Game time.",
		"outro": "Bye.",
	}
	for section, text := range sectionData {
		sectionPath := builder.EpisodeSectionMarkdown(date, section)
		if err := os.WriteFile(sectionPath, []byte(text+"\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", section, err)
		}
	}

	t.Setenv("OPENAI_API_KEY", "sk-test")
	if code := run([]string{"audio", "--date=2025-09-30", "--voice=alloy"}); code != 0 {
		t.Fatalf("audio returned non-zero: %d", code)
	}
	if fake.calls != 5 {
		t.Fatalf("expected 5 TTS calls, got %d", fake.calls)
	}

	mp3Path := builder.EpisodeMP3(date)
	info, err := os.Stat(mp3Path)
	if err != nil {
		t.Fatalf("missing episode.mp3: %v", err)
	}
	expectedSize := int64(len("mp3bytes")*5 + len("pausebytes"))
	if info.Size() != expectedSize {
		t.Fatalf("unexpected episode.mp3 size: %d", info.Size())
	}
}
