package main

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"yodex/internal/ai"
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
	origClient := newTTSClient
	t.Cleanup(func() { newTTSClient = origClient })

	fake := &fakeTTSClient{}
	newTTSClient = func(apiKey string) (ai.TTSClient, error) {
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
	mdPath := builder.EpisodeMarkdown(date)
	if err := os.WriteFile(mdPath, []byte("hello script"), 0o644); err != nil {
		t.Fatalf("write episode.md: %v", err)
	}

	t.Setenv("OPENAI_API_KEY", "sk-test")
	if code := run([]string{"audio", "--date=2025-09-30", "--voice=alloy"}); code != 0 {
		t.Fatalf("audio returned non-zero: %d", code)
	}
	if fake.calls != 1 {
		t.Fatalf("expected 1 TTS call, got %d", fake.calls)
	}

	mp3Path := builder.EpisodeMP3(date)
	info, err := os.Stat(mp3Path)
	if err != nil {
		t.Fatalf("missing episode.mp3: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("episode.mp3 was empty")
	}
}
