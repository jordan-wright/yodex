package podcast

import (
	"context"
	"testing"

	"yodex/internal/config"
)

type fakeTextGen struct {
	called bool
	text   string
	err    error
}

func (f *fakeTextGen) GenerateText(ctx context.Context, model, system, prompt string) (string, error) {
	f.called = true
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
