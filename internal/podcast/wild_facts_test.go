package podcast

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"yodex/internal/config"
	"yodex/internal/storage"
)

func TestGenerateWildFactAvoidsHistoryAndRecordsFact(t *testing.T) {
	history := WildFactHistory{
		Entries: map[string]WildFactEntry{
			"2026-04-19": {Category: "space", Fact: "A day on Venus is longer than its year."},
		},
	}
	data, err := json.Marshal(history)
	if err != nil {
		t.Fatalf("marshal history: %v", err)
	}

	cfg := config.Default()
	cfg.S3Bucket = "wild-fact-bucket"
	cfg.S3Prefix = "yodex"
	fakeStore := &fakeTopicHistoryStore{prefix: cfg.S3Prefix, downloadData: data}
	newWildFactHistoryStore = func(ctx context.Context, cfg config.Config) (wildFactHistoryStore, error) {
		return fakeStore, nil
	}
	t.Cleanup(resetWildFactHistoryStore)

	gen := &fakeBrainBiteGen{responses: []string{"[playful] Octopuses have three hearts. Two pump blood to the gills, while one sends it around the body."}}
	date := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	text, _, err := GenerateWildFactWithUsage(context.Background(), date, cfg, gen, "Volcanoes")
	if err != nil {
		t.Fatalf("GenerateWildFactWithUsage: %v", err)
	}
	if !strings.Contains(text, "three hearts") {
		t.Fatalf("unexpected Wild Fact: %q", text)
	}
	if gen.calls != 1 {
		t.Fatalf("expected one generation call, got %d", gen.calls)
	}
	prompt := gen.prompts[0]
	if !strings.Contains(prompt, "Today's Wild Fact category: music") {
		t.Fatalf("expected deterministic category, got %q", prompt)
	}
	if !strings.Contains(prompt, "Today's main podcast topic: Volcanoes") {
		t.Fatalf("expected daily topic exclusion, got %q", prompt)
	}
	if !strings.Contains(prompt, "Start exactly with \"Daily Fun Fact:\"") {
		t.Fatalf("expected Daily Fun Fact transition guidance, got %q", prompt)
	}
	if !strings.Contains(prompt, "A day on Venus is longer than its year.") {
		t.Fatalf("expected prior fact in prompt, got %q", prompt)
	}

	var saved WildFactHistory
	if err := json.Unmarshal(fakeStore.uploadedData, &saved); err != nil {
		t.Fatalf("parse saved history: %v", err)
	}
	entry := saved.Entries["2026-04-20"]
	if entry.Category != "music" || !strings.Contains(entry.Fact, "three hearts") {
		t.Fatalf("unexpected saved fact: %#v", entry)
	}
	if fakeStore.uploadedKey != "yodex/wild-fact-history.json" {
		t.Fatalf("unexpected upload key: %q", fakeStore.uploadedKey)
	}
}

func TestRecentWildFactsLimitsNewestEntries(t *testing.T) {
	history := WildFactHistory{
		Entries: map[string]WildFactEntry{
			"2026-04-19": {Fact: "Older"},
			"2026-04-20": {Fact: "Newer"},
		},
	}
	recent := recentWildFacts(history, 1)
	if len(recent) != 1 || recent[0].Fact != "Newer" {
		t.Fatalf("unexpected recent facts: %#v", recent)
	}
}

func resetWildFactHistoryStore() {
	newWildFactHistoryStore = func(ctx context.Context, cfg config.Config) (wildFactHistoryStore, error) {
		return storage.New(ctx, cfg.S3Bucket, cfg.S3Prefix, cfg.Region)
	}
}
