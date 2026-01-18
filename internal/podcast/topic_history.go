package podcast

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"yodex/internal/config"
	"yodex/internal/storage"
)

// TopicHistoryEntry tracks recent topics to avoid repetition.
type TopicHistoryEntry struct {
	Topic string `json:"topic"`
}

// TopicHistory is the manifest stored alongside episodes.
type TopicHistory struct {
	Entries map[string]TopicHistoryEntry `json:"entries"`
}

type topicHistoryStore interface {
	DownloadBytes(ctx context.Context, key string) ([]byte, error)
	UploadBytes(ctx context.Context, key string, data []byte, contentType, cacheControl string) error
	Prefix() string
}

var newTopicHistoryStore = func(ctx context.Context, cfg config.Config) (topicHistoryStore, error) {
	return storage.New(ctx, cfg.S3Bucket, cfg.S3Prefix, cfg.Region)
}

func loadTopicHistory(ctx context.Context, cfg config.Config) (TopicHistory, error) {
	if cfg.S3Bucket != "" {
		return loadTopicHistoryFromS3(ctx, cfg)
	}
	return loadTopicHistoryFromFile(cfg)
}

func saveTopicHistory(ctx context.Context, cfg config.Config, history TopicHistory) error {
	if cfg.S3Bucket != "" {
		return saveTopicHistoryToS3(ctx, cfg, history)
	}
	return saveTopicHistoryToFile(cfg, history)
}

func trimTopicHistory(history TopicHistory, size int) TopicHistory {
	if size <= 0 {
		return TopicHistory{Entries: map[string]TopicHistoryEntry{}}
	}
	if len(history.Entries) <= size {
		return history
	}
	keys := sortedHistoryKeys(history)
	keep := keys[len(keys)-size:]
	trimmed := TopicHistory{Entries: make(map[string]TopicHistoryEntry, len(keep))}
	for _, key := range keep {
		trimmed.Entries[key] = history.Entries[key]
	}
	return trimmed
}

func appendTopicHistory(ctx context.Context, cfg config.Config, history TopicHistory, date time.Time, entry TopicHistoryEntry) error {
	if cfg.TopicHistorySize <= 0 {
		return nil
	}
	if history.Entries == nil {
		history.Entries = make(map[string]TopicHistoryEntry)
	}
	dateKey := date.UTC().Format("2006-01-02")
	history.Entries[dateKey] = entry
	history = trimTopicHistory(history, cfg.TopicHistorySize)
	return saveTopicHistory(ctx, cfg, history)
}

func loadTopicHistoryFromS3(ctx context.Context, cfg config.Config) (TopicHistory, error) {
	store, err := newTopicHistoryStore(ctx, cfg)
	if err != nil {
		slog.Warn("failed to initialize topic history store", "err", err)
		return TopicHistory{Entries: map[string]TopicHistoryEntry{}}, nil
	}
	key := topicHistoryKey(store)
	data, err := store.DownloadBytes(ctx, key)
	if err != nil {
		slog.Warn("failed to load topic history from s3", "key", key, "err", err)
		return TopicHistory{Entries: map[string]TopicHistoryEntry{}}, nil
	}
	var history TopicHistory
	if err := json.Unmarshal(data, &history); err != nil {
		slog.Warn("failed to parse topic history from s3", "key", key, "err", err)
		return TopicHistory{Entries: map[string]TopicHistoryEntry{}}, nil
	}
	if history.Entries == nil {
		history.Entries = map[string]TopicHistoryEntry{}
	}
	return history, nil
}

func saveTopicHistoryToS3(ctx context.Context, cfg config.Config, history TopicHistory) error {
	store, err := newTopicHistoryStore(ctx, cfg)
	if err != nil {
		return err
	}
	key := topicHistoryKey(store)
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	if err := store.UploadBytes(ctx, key, data, "application/json", "no-cache"); err != nil {
		return fmt.Errorf("upload topic history: %w", err)
	}
	return nil
}

func topicHistoryKey(store topicHistoryStore) string {
	prefix := store.Prefix()
	if prefix == "" {
		return "topic-history.json"
	}
	return path.Join(prefix, "topic-history.json")
}

func loadTopicHistoryFromFile(cfg config.Config) (TopicHistory, error) {
	if strings.TrimSpace(cfg.TopicHistoryPath) == "" {
		return TopicHistory{Entries: map[string]TopicHistoryEntry{}}, nil
	}
	data, err := os.ReadFile(cfg.TopicHistoryPath)
	if err != nil {
		slog.Warn("failed to read topic history file", "path", cfg.TopicHistoryPath, "err", err)
		return TopicHistory{Entries: map[string]TopicHistoryEntry{}}, nil
	}
	var history TopicHistory
	if err := json.Unmarshal(data, &history); err != nil {
		slog.Warn("failed to parse topic history file", "path", cfg.TopicHistoryPath, "err", err)
		return TopicHistory{Entries: map[string]TopicHistoryEntry{}}, nil
	}
	if history.Entries == nil {
		history.Entries = map[string]TopicHistoryEntry{}
	}
	return history, nil
}

func saveTopicHistoryToFile(cfg config.Config, history TopicHistory) error {
	if strings.TrimSpace(cfg.TopicHistoryPath) == "" {
		return nil
	}
	if history.Entries == nil {
		history.Entries = map[string]TopicHistoryEntry{}
	}
	if err := os.MkdirAll(filepath.Dir(cfg.TopicHistoryPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfg.TopicHistoryPath, data, 0o644)
}

func sortedHistoryKeys(history TopicHistory) []string {
	keys := make([]string, 0, len(history.Entries))
	for key := range history.Entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func recentTopics(history TopicHistory) []string {
	keys := sortedHistoryKeys(history)
	topics := make([]string, 0, len(keys))
	for i := len(keys) - 1; i >= 0; i-- {
		entry := history.Entries[keys[i]]
		if strings.TrimSpace(entry.Topic) == "" {
			continue
		}
		topics = append(topics, strings.TrimSpace(entry.Topic))
	}
	return topics
}
