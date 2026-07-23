package podcast

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"yodex/internal/config"
	"yodex/internal/storage"
)

// BrainBiteCoverage tracks what a daily Brain Bite already covered.
type BrainBiteCoverage struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// BrainBiteWeek tracks the weekly Brain Bite theme and daily coverage.
type BrainBiteWeek struct {
	WeekStart string              `json:"weekStart"`
	Topic     string              `json:"topic"`
	Subtopics []string            `json:"subtopics,omitempty"`
	Coverage  []BrainBiteCoverage `json:"coverage"`
}

// BrainBiteHistory is the manifest stored alongside episodes.
type BrainBiteHistory struct {
	Weeks map[string]BrainBiteWeek `json:"weeks"`
}

type brainBiteHistoryStore interface {
	DownloadBytes(ctx context.Context, key string) ([]byte, error)
	UploadBytes(ctx context.Context, key string, data []byte, contentType, cacheControl string) error
	Prefix() string
}

var newBrainBiteHistoryStore = func(ctx context.Context, cfg config.Config) (brainBiteHistoryStore, error) {
	return storage.New(ctx, cfg.S3Bucket, cfg.S3Prefix, cfg.Region)
}

func loadBrainBiteHistory(ctx context.Context, cfg config.Config) (BrainBiteHistory, error) {
	if cfg.S3Bucket != "" {
		return loadBrainBiteHistoryFromS3(ctx, cfg)
	}
	return loadBrainBiteHistoryFromFile(cfg)
}

func saveBrainBiteHistory(ctx context.Context, cfg config.Config, history BrainBiteHistory) error {
	if cfg.S3Bucket != "" {
		return saveBrainBiteHistoryToS3(ctx, cfg, history)
	}
	return saveBrainBiteHistoryToFile(cfg, history)
}

func loadBrainBiteHistoryFromS3(ctx context.Context, cfg config.Config) (BrainBiteHistory, error) {
	store, err := newBrainBiteHistoryStore(ctx, cfg)
	if err != nil {
		return BrainBiteHistory{}, fmt.Errorf("initialize brain bite history store: %w", err)
	}
	key := brainBiteHistoryKey(store)
	data, err := store.DownloadBytes(ctx, key)
	if err != nil {
		if storage.IsNotFound(err) {
			return BrainBiteHistory{Weeks: map[string]BrainBiteWeek{}}, nil
		}
		return BrainBiteHistory{}, fmt.Errorf("download brain bite history %q: %w", key, err)
	}
	var history BrainBiteHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return BrainBiteHistory{}, fmt.Errorf("parse brain bite history %q: %w", key, err)
	}
	if history.Weeks == nil {
		history.Weeks = map[string]BrainBiteWeek{}
	}
	return history, nil
}

func saveBrainBiteHistoryToS3(ctx context.Context, cfg config.Config, history BrainBiteHistory) error {
	store, err := newBrainBiteHistoryStore(ctx, cfg)
	if err != nil {
		return err
	}
	key := brainBiteHistoryKey(store)
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	if err := store.UploadBytes(ctx, key, data, "application/json", "no-cache"); err != nil {
		return fmt.Errorf("upload brain bite history: %w", err)
	}
	return nil
}

func brainBiteHistoryKey(store brainBiteHistoryStore) string {
	prefix := store.Prefix()
	if prefix == "" {
		return "brain-bite-history.json"
	}
	return path.Join(prefix, "brain-bite-history.json")
}

func loadBrainBiteHistoryFromFile(cfg config.Config) (BrainBiteHistory, error) {
	if strings.TrimSpace(cfg.BrainBiteHistoryPath) == "" {
		return BrainBiteHistory{Weeks: map[string]BrainBiteWeek{}}, nil
	}
	data, err := os.ReadFile(cfg.BrainBiteHistoryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return BrainBiteHistory{Weeks: map[string]BrainBiteWeek{}}, nil
		}
		return BrainBiteHistory{}, fmt.Errorf("read brain bite history file %q: %w", cfg.BrainBiteHistoryPath, err)
	}
	var history BrainBiteHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return BrainBiteHistory{}, fmt.Errorf("parse brain bite history file %q: %w", cfg.BrainBiteHistoryPath, err)
	}
	if history.Weeks == nil {
		history.Weeks = map[string]BrainBiteWeek{}
	}
	return history, nil
}

func saveBrainBiteHistoryToFile(cfg config.Config, history BrainBiteHistory) error {
	if strings.TrimSpace(cfg.BrainBiteHistoryPath) == "" {
		return nil
	}
	if history.Weeks == nil {
		history.Weeks = map[string]BrainBiteWeek{}
	}
	if err := os.MkdirAll(filepath.Dir(cfg.BrainBiteHistoryPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfg.BrainBiteHistoryPath, data, 0o644)
}

func previousBrainBiteTopics(history BrainBiteHistory) []string {
	keys := make([]string, 0, len(history.Weeks))
	for key := range history.Weeks {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	topics := make([]string, 0, len(keys))
	for i := len(keys) - 1; i >= 0; i-- {
		topic := strings.TrimSpace(history.Weeks[keys[i]].Topic)
		if topic == "" {
			continue
		}
		topics = append(topics, topic)
	}
	return topics
}
