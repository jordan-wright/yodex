package main

import (
	"context"
	"os"
	"testing"
	"time"

	"yodex/internal/ai"
	"yodex/internal/paths"
)

func TestHelp(t *testing.T) {
	if code := run([]string{"-h"}); code != 0 {
		t.Fatalf("expected help to return 0, got %d", code)
	}
}

func TestUnknownSubcommand(t *testing.T) {
	if code := run([]string{"unknown"}); code == 0 {
		t.Fatalf("expected non-zero for unknown subcommand")
	}
}

func TestScriptFlagParsing(t *testing.T) {
	origClient := newTextClient
	t.Cleanup(func() { newTextClient = origClient })

	fake := &fakeTextClient{
		responses: makeSectionResponses(800),
	}
	newTextClient = func(apiKey string) (ai.TextClient, error) {
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

	t.Setenv("OPENAI_API_KEY", "sk-test")
	if code := run([]string{"script", "--date=2025-09-30", "--log-level=debug", "--topic=Test Topic"}); code != 0 {
		t.Fatalf("script returned non-zero: %d", code)
	}
}

func TestAudioFlagParsing(t *testing.T) {
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
}

func TestPublishFlagParsing(t *testing.T) {
	origUploader := newUploader
	t.Cleanup(func() { newUploader = origUploader })

	newUploader = func(ctx context.Context, bucket, prefix, region string) (uploader, error) {
		return &fakeUploader{}, nil
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
	if err := os.WriteFile(builder.EpisodeMP3(date), []byte("audio"), 0o644); err != nil {
		t.Fatalf("write mp3: %v", err)
	}

	if code := run([]string{"publish", "--date=2025-09-30", "--bucket=b", "--prefix=yodex", "--region=us-east-1"}); code != 0 {
		t.Fatalf("publish returned non-zero: %d", code)
	}
}

func TestTopicFlagParsing(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := tmp + "/config.json"
	if err := os.WriteFile(cfgPath, []byte(`{"topic":"Test Topic"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if code := run([]string{"topic", "--config", cfgPath}); code != 0 {
		t.Fatalf("topic returned non-zero: %d", code)
	}
}
