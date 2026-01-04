package podcast

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"yodex/internal/config"
	"yodex/internal/storage"
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

type fakeTopicHistoryStore struct {
	prefix       string
	downloadData []byte
	downloadErr  error
	uploadedKey  string
	uploadedData []byte
}

func (f *fakeTopicHistoryStore) DownloadBytes(ctx context.Context, key string) ([]byte, error) {
	if f.downloadErr != nil {
		return nil, f.downloadErr
	}
	return f.downloadData, nil
}

func (f *fakeTopicHistoryStore) UploadBytes(ctx context.Context, key string, data []byte, contentType, cacheControl string) error {
	f.uploadedKey = key
	f.uploadedData = data
	return nil
}

func (f *fakeTopicHistoryStore) Prefix() string {
	return f.prefix
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
	history := TopicHistory{
		Entries: []TopicHistoryEntry{
			{Topic: "Ocean Life", PublishedAt: time.Now().Add(-48 * time.Hour)},
			{Topic: "Volcanoes", PublishedAt: time.Now().Add(-24 * time.Hour)},
		},
	}
	data, err := json.Marshal(history)
	if err != nil {
		t.Fatalf("marshal history: %v", err)
	}

	cfg := config.Default()
	cfg.TopicHistoryS3Prefix = "topic-history"
	cfg.TopicHistorySize = 3
	fakeStore := &fakeTopicHistoryStore{
		prefix:       cfg.TopicHistoryS3Prefix,
		downloadData: data,
	}
	newTopicHistoryStore = func(ctx context.Context, cfg config.Config) (topicHistoryStore, error) {
		return fakeStore, nil
	}
	t.Cleanup(func() {
		newTopicHistoryStore = func(ctx context.Context, cfg config.Config) (topicHistoryStore, error) {
			return storage.New(ctx, cfg.S3Bucket, cfg.TopicHistoryS3Prefix, cfg.Region)
		}
	})
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
	cfg := config.Default()
	cfg.TopicHistoryS3Prefix = "topic-history"
	cfg.TopicHistorySize = 3
	fakeStore := &fakeTopicHistoryStore{
		prefix:      cfg.TopicHistoryS3Prefix,
		downloadErr: &types.NoSuchKey{},
	}
	newTopicHistoryStore = func(ctx context.Context, cfg config.Config) (topicHistoryStore, error) {
		return fakeStore, nil
	}
	t.Cleanup(func() {
		newTopicHistoryStore = func(ctx context.Context, cfg config.Config) (topicHistoryStore, error) {
			return storage.New(ctx, cfg.S3Bucket, cfg.TopicHistoryS3Prefix, cfg.Region)
		}
	})
	gen := &fakeTextGen{text: "Desert Animals"}
	if _, err := SelectTopic(context.Background(), cfg, gen); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(gen.prompt, "Recent topics") {
		t.Fatalf("expected no history prompt, got %q", gen.prompt)
	}
}

func TestSelectTopicHistoryTruncation(t *testing.T) {
	history := TopicHistory{
		Entries: []TopicHistoryEntry{
			{Topic: "Rainforests", PublishedAt: time.Now().Add(-72 * time.Hour)},
			{Topic: "Dinosaurs", PublishedAt: time.Now().Add(-48 * time.Hour)},
			{Topic: "Gravity", PublishedAt: time.Now().Add(-24 * time.Hour)},
		},
	}
	data, err := json.Marshal(history)
	if err != nil {
		t.Fatalf("marshal history: %v", err)
	}

	cfg := config.Default()
	cfg.TopicHistoryS3Prefix = "topic-history"
	cfg.TopicHistorySize = 2
	fakeStore := &fakeTopicHistoryStore{
		prefix:       cfg.TopicHistoryS3Prefix,
		downloadData: data,
	}
	newTopicHistoryStore = func(ctx context.Context, cfg config.Config) (topicHistoryStore, error) {
		return fakeStore, nil
	}
	t.Cleanup(func() {
		newTopicHistoryStore = func(ctx context.Context, cfg config.Config) (topicHistoryStore, error) {
			return storage.New(ctx, cfg.S3Bucket, cfg.TopicHistoryS3Prefix, cfg.Region)
		}
	})
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
