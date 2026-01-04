package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"yodex/internal/ai"
	"yodex/internal/paths"
	"yodex/internal/podcast"
)

type fakeTextClient struct {
	responses []string
	calls     int
}

func (f *fakeTextClient) GenerateText(ctx context.Context, model, system, prompt string) (string, error) {
	if f.calls >= len(f.responses) {
		f.calls++
		return "", nil
	}
	resp := f.responses[f.calls]
	f.calls++
	return resp, nil
}

func (f *fakeTextClient) GenerateTextWithUsage(ctx context.Context, model, system, prompt string) (string, ai.TokenUsage, error) {
	text, err := f.GenerateText(ctx, model, system, prompt)
	return text, ai.TokenUsage{}, err
}

func TestScriptWritesOutputs(t *testing.T) {
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
	if code := run([]string{"script", "--date=2025-09-30", "--topic=Test Topic"}); code != 0 {
		t.Fatalf("script returned non-zero: %d", code)
	}
	if fake.calls != 4 {
		t.Fatalf("expected 4 AI calls, got %d", fake.calls)
	}

	date := time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC)
	builder := paths.New("")
	mdPath := builder.EpisodeMarkdown(date)
	metaPath := builder.EpisodeMeta(date)

	if _, err := os.Stat(mdPath); err != nil {
		t.Fatalf("missing episode.md: %v", err)
	}
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read meta.json: %v", err)
	}
	var meta scriptMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatalf("parse meta.json: %v", err)
	}
	if meta.Topic != "Test Topic" {
		t.Fatalf("meta topic mismatch: %s", meta.Topic)
	}
	if meta.WordCount <= 0 {
		t.Fatalf("meta wordCount missing: %d", meta.WordCount)
	}
}

func TestScriptAcceptsShorterScripts(t *testing.T) {
	origClient := newTextClient
	t.Cleanup(func() { newTextClient = origClient })

	fake := &fakeTextClient{
		responses: makeSectionResponses(100),
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
	if code := run([]string{"script", "--date=2025-09-30", "--topic=Retry Topic"}); code != 0 {
		t.Fatalf("script returned non-zero: %d", code)
	}
	if fake.calls != 4 {
		t.Fatalf("expected 4 AI calls, got %d", fake.calls)
	}
}

func makeSectionResponses(targetWords int) []string {
	ep := podcast.Episode{
		Title: "Test Title",
		Sections: []podcast.EpisodeSection{
			{SectionID: "intro", Text: "Intro text."},
			{SectionID: "core-idea", Text: "Core idea text."},
			{SectionID: "deep-dive", Text: ""},
			{SectionID: "outro", Text: "Recap text. What did you learn?"},
		},
	}
	baseWords := podcast.WordCount(ep.RenderMarkdown())
	remaining := targetWords - baseWords
	if remaining < 1 {
		remaining = 1
	}
	ep.Sections[2].Text = makeWordyText(remaining)
	return []string{
		ep.Sections[0].Text,
		ep.Sections[1].Text,
		ep.Sections[2].Text,
		ep.Sections[3].Text,
	}
}

func makeWordyText(targetWords int) string {
	var b strings.Builder
	for i := 0; i < targetWords; i++ {
		b.WriteString("word")
		if i < targetWords-1 {
			b.WriteString(" ")
		}
	}
	return b.String()
}
