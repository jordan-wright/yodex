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

// WildFactEntry records a previously generated Wild Fact.
type WildFactEntry struct {
	Category string `json:"category"`
	Fact     string `json:"fact"`
}

// WildFactHistory is the manifest used to avoid repeating Wild Facts.
type WildFactHistory struct {
	Entries map[string]WildFactEntry `json:"entries"`
}

type wildFactHistoryStore interface {
	DownloadBytes(ctx context.Context, key string) ([]byte, error)
	UploadBytes(ctx context.Context, key string, data []byte, contentType, cacheControl string) error
	Prefix() string
}

var newWildFactHistoryStore = func(ctx context.Context, cfg config.Config) (wildFactHistoryStore, error) {
	return storage.New(ctx, cfg.S3Bucket, cfg.S3Prefix, cfg.Region)
}

func loadWildFactHistory(ctx context.Context, cfg config.Config) (WildFactHistory, error) {
	if cfg.S3Bucket != "" {
		return loadWildFactHistoryFromS3(ctx, cfg)
	}
	return loadWildFactHistoryFromFile(cfg)
}

func saveWildFactHistory(ctx context.Context, cfg config.Config, history WildFactHistory) error {
	if cfg.S3Bucket != "" {
		return saveWildFactHistoryToS3(ctx, cfg, history)
	}
	return saveWildFactHistoryToFile(cfg, history)
}

func loadWildFactHistoryFromS3(ctx context.Context, cfg config.Config) (WildFactHistory, error) {
	store, err := newWildFactHistoryStore(ctx, cfg)
	if err != nil {
		return WildFactHistory{}, fmt.Errorf("initialize wild fact history store: %w", err)
	}
	key := wildFactHistoryKey(store)
	data, err := store.DownloadBytes(ctx, key)
	if err != nil {
		if storage.IsNotFound(err) {
			return WildFactHistory{Entries: map[string]WildFactEntry{}}, nil
		}
		return WildFactHistory{}, fmt.Errorf("download wild fact history %q: %w", key, err)
	}
	return parseWildFactHistory(data, fmt.Sprintf("S3 object %q", key))
}

func saveWildFactHistoryToS3(ctx context.Context, cfg config.Config, history WildFactHistory) error {
	store, err := newWildFactHistoryStore(ctx, cfg)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	if err := store.UploadBytes(ctx, wildFactHistoryKey(store), data, "application/json", "no-cache"); err != nil {
		return fmt.Errorf("upload wild fact history: %w", err)
	}
	return nil
}

func wildFactHistoryKey(store wildFactHistoryStore) string {
	if store.Prefix() == "" {
		return "wild-fact-history.json"
	}
	return path.Join(store.Prefix(), "wild-fact-history.json")
}

func loadWildFactHistoryFromFile(cfg config.Config) (WildFactHistory, error) {
	if strings.TrimSpace(cfg.WildFactHistoryPath) == "" {
		return WildFactHistory{Entries: map[string]WildFactEntry{}}, nil
	}
	data, err := os.ReadFile(cfg.WildFactHistoryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return WildFactHistory{Entries: map[string]WildFactEntry{}}, nil
		}
		return WildFactHistory{}, fmt.Errorf("read wild fact history file %q: %w", cfg.WildFactHistoryPath, err)
	}
	return parseWildFactHistory(data, fmt.Sprintf("file %q", cfg.WildFactHistoryPath))
}

func saveWildFactHistoryToFile(cfg config.Config, history WildFactHistory) error {
	if strings.TrimSpace(cfg.WildFactHistoryPath) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(cfg.WildFactHistoryPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfg.WildFactHistoryPath, data, 0o644)
}

func parseWildFactHistory(data []byte, source string) (WildFactHistory, error) {
	var history WildFactHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return WildFactHistory{}, fmt.Errorf("parse wild fact history from %s: %w", source, err)
	}
	if history.Entries == nil {
		history.Entries = map[string]WildFactEntry{}
	}
	return history, nil
}

func recentWildFacts(history WildFactHistory, limit int) []WildFactEntry {
	keys := make([]string, 0, len(history.Entries))
	for key := range history.Entries {
		keys = append(keys, key)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	if limit > 0 && len(keys) > limit {
		keys = keys[:limit]
	}
	entries := make([]WildFactEntry, 0, len(keys))
	for _, key := range keys {
		entry := history.Entries[key]
		if strings.TrimSpace(entry.Fact) != "" {
			entries = append(entries, entry)
		}
	}
	return entries
}
