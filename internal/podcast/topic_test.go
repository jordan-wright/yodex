package podcast

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"yodex/internal/config"
)

type fakeTextGen struct {
	called bool
	text   string
	err    error
	prompt string
}

func (f *fakeTextGen) GenerateText(ctx context.Context, model, system, prompt string) (string, error) {
	f.called = true
	f.prompt = prompt
	return f.text, f.err
}

func TestSelectTopicUsesConfig(t *testing.T) {
	cfg := config.Default()
	cfg.Topic = "Configured Topic"

	gen := &fakeTextGen{text: "Generated Topic"}
	topic, err := SelectTopic(context.Background(), cfg, gen)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if topic != "Configured Topic" {
		t.Fatalf("expected configured topic, got %q", topic)
	}
	if gen.called {
		t.Fatalf("expected generator not to be called")
	}
}

func TestSelectTopicGenerates(t *testing.T) {
	cfg := config.Default()
	gen := &fakeTextGen{text: "Ocean Wonders\n"}
	topic, err := SelectTopic(context.Background(), cfg, gen)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if topic != "Ocean Wonders" {
		t.Fatalf("unexpected topic: %q", topic)
	}
	if !gen.called {
		t.Fatalf("expected generator to be called")
	}
}

func TestSelectTopicIncludesHistoryInPrompt(t *testing.T) {
	tmp := t.TempDir()
	historyPath := filepath.Join(tmp, "topic-history.json")
	history := TopicHistory{
		Entries: []TopicHistoryEntry{
			{Topic: "Ocean Life", PublishedAt: time.Now().Add(-48 * time.Hour)},
			{Topic: "Volcanoes", PublishedAt: time.Now().Add(-24 * time.Hour)},
		},
	}
	if err := saveTopicHistory(historyPath, history); err != nil {
		t.Fatalf("save history: %v", err)
	}

	cfg := config.Default()
	cfg.TopicHistoryPath = historyPath
	cfg.TopicHistorySize = 3
	gen := &fakeTextGen{text: "Space Weather"}
	if _, err := SelectTopic(context.Background(), cfg, gen); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(gen.prompt, "Recent topics (do not repeat or closely paraphrase any topics in this list):") {
		t.Fatalf("expected recent topics instruction in prompt")
	}
	if !strings.Contains(gen.prompt, "- Ocean Life") || !strings.Contains(gen.prompt, "- Volcanoes") {
		t.Fatalf("expected history list in prompt, got %q", gen.prompt)
	}
}

func TestSelectTopicEmptyHistory(t *testing.T) {
	tmp := t.TempDir()
	historyPath := filepath.Join(tmp, "missing-history.json")

	cfg := config.Default()
	cfg.TopicHistoryPath = historyPath
	cfg.TopicHistorySize = 3
	gen := &fakeTextGen{text: "Desert Animals"}
	if _, err := SelectTopic(context.Background(), cfg, gen); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(gen.prompt, "Recent topics") {
		t.Fatalf("expected no history prompt, got %q", gen.prompt)
	}
	if _, err := os.Stat(historyPath); err != nil {
		t.Fatalf("expected history file to be created: %v", err)
	}
}

func TestSelectTopicHistoryTruncation(t *testing.T) {
	tmp := t.TempDir()
	historyPath := filepath.Join(tmp, "topic-history.json")
	history := TopicHistory{
		Entries: []TopicHistoryEntry{
			{Topic: "Rainforests", PublishedAt: time.Now().Add(-72 * time.Hour)},
			{Topic: "Dinosaurs", PublishedAt: time.Now().Add(-48 * time.Hour)},
			{Topic: "Gravity", PublishedAt: time.Now().Add(-24 * time.Hour)},
		},
	}
	if err := saveTopicHistory(historyPath, history); err != nil {
		t.Fatalf("save history: %v", err)
	}

	cfg := config.Default()
	cfg.TopicHistoryPath = historyPath
	cfg.TopicHistorySize = 2
	gen := &fakeTextGen{text: "Coral Reefs"}
	if _, err := SelectTopic(context.Background(), cfg, gen); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(gen.prompt, "- Gravity") {
		t.Fatalf("expected prompt to truncate history, got %q", gen.prompt)
	}
	if !strings.Contains(gen.prompt, "- Rainforests") || !strings.Contains(gen.prompt, "- Dinosaurs") {
		t.Fatalf("expected prompt to include latest history items, got %q", gen.prompt)
	}
}
