package podcast

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
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

func loadTopicHistory(path string) (TopicHistory, error) {
	if path == "" {
		return TopicHistory{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
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

func saveTopicHistory(path string, history TopicHistory) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
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

func appendTopicHistory(path string, size int, history TopicHistory, entry TopicHistoryEntry) error {
	if size <= 0 || path == "" {
		return nil
	}
	history.Entries = append([]TopicHistoryEntry{entry}, history.Entries...)
	history = trimTopicHistory(history, size)
	return saveTopicHistory(path, history)
}
