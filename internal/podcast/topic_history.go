package podcast

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"yodex/internal/config"
	"yodex/internal/storage"
)

// TopicHistoryEntry tracks recent topics to avoid repetition.
type TopicHistoryEntry struct {
	Topic       string    `json:"topic"`
	EpisodeID   string    `json:"episodeId,omitempty"`
	PublishedAt time.Time `json:"publishedAt"`
}

// TopicHistory is the manifest stored alongside episodes.
type TopicHistory struct {
	Entries []TopicHistoryEntry `json:"entries"`
}

type topicHistoryStore interface {
	DownloadBytes(ctx context.Context, key string) ([]byte, error)
	UploadBytes(ctx context.Context, key string, data []byte, contentType, cacheControl string) error
	Prefix() string
}

var newTopicHistoryStore = func(ctx context.Context, cfg config.Config) (topicHistoryStore, error) {
	return storage.New(ctx, cfg.S3Bucket, cfg.TopicHistoryS3Prefix, cfg.Region)
}

func loadTopicHistory(ctx context.Context, cfg config.Config) (TopicHistory, error) {
	if cfg.TopicHistoryS3Prefix != "" {
		return loadTopicHistoryFromS3(ctx, cfg)
	}
	return TopicHistory{}, nil
}

func saveTopicHistory(ctx context.Context, cfg config.Config, history TopicHistory) error {
	if cfg.TopicHistoryS3Prefix != "" {
		return saveTopicHistoryToS3(ctx, cfg, history)
	}
	return nil
}

func trimTopicHistory(history TopicHistory, size int) TopicHistory {
	if size <= 0 {
		return TopicHistory{}
	}
	if len(history.Entries) > size {
		history.Entries = history.Entries[:size]
	}
	return history
}

func appendTopicHistory(ctx context.Context, cfg config.Config, history TopicHistory, entry TopicHistoryEntry) error {
	if cfg.TopicHistorySize <= 0 {
		return nil
	}
	if cfg.TopicHistoryS3Prefix == "" {
		return nil
	}
	history.Entries = append([]TopicHistoryEntry{entry}, history.Entries...)
	history = trimTopicHistory(history, cfg.TopicHistorySize)
	return saveTopicHistory(ctx, cfg, history)
}

func loadTopicHistoryFromS3(ctx context.Context, cfg config.Config) (TopicHistory, error) {
	store, err := newTopicHistoryStore(ctx, cfg)
	if err != nil {
		return TopicHistory{}, err
	}
	key := topicHistoryKey(store)
	data, err := store.DownloadBytes(ctx, key)
	if err != nil {
		if storage.IsNotFound(err) {
			return TopicHistory{}, nil
		}
		return TopicHistory{}, err
	}
	var history TopicHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return TopicHistory{}, err
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
