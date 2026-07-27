package podcast

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"yodex/internal/config"
	"yodex/internal/storage"
)

type fakeBrainBiteGen struct {
	responses []string
	prompts   []string
	calls     int
}

func (f *fakeBrainBiteGen) GenerateText(ctx context.Context, model, system, prompt string) (string, error) {
	f.prompts = append(f.prompts, prompt)
	if f.calls >= len(f.responses) {
		f.calls++
		return "", nil
	}
	text := f.responses[f.calls]
	f.calls++
	return text, nil
}

func TestGenerateBrainBiteCreatesWeekWithPreviousTopics(t *testing.T) {
	history := BrainBiteHistory{
		Weeks: map[string]BrainBiteWeek{
			"2026-W14": {WeekStart: "2026-03-30", Topic: "Different Kinds of Bugs"},
			"2026-W15": {WeekStart: "2026-04-06", Topic: "A Tour of Our Solar System"},
		},
	}
	data, err := json.Marshal(history)
	if err != nil {
		t.Fatalf("marshal history: %v", err)
	}

	cfg := config.Default()
	cfg.S3Bucket = "brain-bite-bucket"
	cfg.S3Prefix = "yodex"
	fakeStore := &fakeTopicHistoryStore{
		prefix:       cfg.S3Prefix,
		downloadData: data,
	}
	newBrainBiteHistoryStore = func(ctx context.Context, cfg config.Config) (brainBiteHistoryStore, error) {
		return fakeStore, nil
	}
	t.Cleanup(func() {
		newBrainBiteHistoryStore = func(ctx context.Context, cfg config.Config) (brainBiteHistoryStore, error) {
			return storage.New(ctx, cfg.S3Bucket, cfg.S3Prefix, cfg.Region)
		}
	})

	gen := &fakeBrainBiteGen{
		responses: []string{
			"Animal Homes",
			"Nests\nBurrows\nHives\nWebs\nShells and borrowed homes\nUnderwater hideouts\nHow animal homes match animal needs",
			"Today, our Brain Bite begins with nests. A nest can protect eggs and babies.",
		},
	}
	text, _, err := GenerateBrainBiteWithUsage(context.Background(), time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC), cfg, gen)
	if err != nil {
		t.Fatalf("GenerateBrainBiteWithUsage: %v", err)
	}
	if !strings.Contains(text, "nests") {
		t.Fatalf("unexpected brain bite text: %q", text)
	}
	if gen.calls != 3 {
		t.Fatalf("expected topic, plan, and lesson calls, got %d", gen.calls)
	}
	if !strings.Contains(gen.prompts[0], "Previous Brain Bite weekly topics") {
		t.Fatalf("expected previous topics prompt, got %q", gen.prompts[0])
	}
	if !strings.Contains(gen.prompts[0], "- Different Kinds of Bugs") || !strings.Contains(gen.prompts[0], "- A Tour of Our Solar System") {
		t.Fatalf("expected previous topic titles in prompt, got %q", gen.prompts[0])
	}
	if !strings.Contains(gen.prompts[1], "Return exactly seven short lesson titles") {
		t.Fatalf("expected weekly plan prompt, got %q", gen.prompts[1])
	}
	if !strings.Contains(gen.prompts[2], "Do not greet the audience again") {
		t.Fatalf("expected anti-reintro guidance, got %q", gen.prompts[2])
	}
	if !strings.Contains(gen.prompts[2], "gentle transition sentence") || !strings.Contains(gen.prompts[2], "it's time for the daily Brain Bite") {
		t.Fatalf("expected named Brain Bite bridge guidance, got %q", gen.prompts[2])
	}
	if !strings.Contains(gen.prompts[2], "do not assume the listener gave a particular answer") || !strings.Contains(gen.prompts[2], "repeat game details") {
		t.Fatalf("expected neutral listener-response guidance, got %q", gen.prompts[2])
	}
	if !strings.Contains(gen.prompts[2], "Tomorrow's planned subtopic: Burrows") {
		t.Fatalf("expected planned tomorrow subtopic, got %q", gen.prompts[2])
	}
	if fakeStore.uploadedKey != "yodex/brain-bite-history.json" {
		t.Fatalf("unexpected upload key: %s", fakeStore.uploadedKey)
	}
	var saved BrainBiteHistory
	if err := json.Unmarshal(fakeStore.uploadedData, &saved); err != nil {
		t.Fatalf("parse saved history: %v", err)
	}
	week := saved.Weeks["2026-W17"]
	if week.Topic != "Animal Homes" {
		t.Fatalf("unexpected week topic: %q", week.Topic)
	}
	if len(week.Subtopics) != 7 || week.Subtopics[0] != "Nests" || week.Subtopics[1] != "Burrows" {
		t.Fatalf("unexpected week subtopics: %#v", week.Subtopics)
	}
	if week.WeekStart != "2026-04-20" {
		t.Fatalf("unexpected week start: %q", week.WeekStart)
	}
	if len(week.Coverage) != 1 || week.Coverage[0].Date != "2026-04-20" {
		t.Fatalf("expected saved daily coverage, got %#v", week.Coverage)
	}
	if week.Coverage[0].Title != "Nests" {
		t.Fatalf("expected coverage title from planned subtopic, got %#v", week.Coverage[0])
	}
}

func TestGenerateBrainBiteUsesExistingWeekCoverage(t *testing.T) {
	history := BrainBiteHistory{
		Weeks: map[string]BrainBiteWeek{
			"2026-W17": {
				WeekStart: "2026-04-20",
				Topic:     "Animal Homes",
				Subtopics: []string{"Nests", "Burrows", "Hives", "Webs", "Shells", "Underwater hideouts", "Recap"},
				Coverage: []BrainBiteCoverage{
					{
						Date:    "2026-04-20",
						Title:   "Nests",
						Summary: "Introduced nests as animal homes that protect eggs and babies.",
					},
				},
			},
		},
	}
	data, err := json.Marshal(history)
	if err != nil {
		t.Fatalf("marshal history: %v", err)
	}

	cfg := config.Default()
	cfg.S3Bucket = "brain-bite-bucket"
	cfg.S3Prefix = "yodex"
	fakeStore := &fakeTopicHistoryStore{
		prefix:       cfg.S3Prefix,
		downloadData: data,
	}
	newBrainBiteHistoryStore = func(ctx context.Context, cfg config.Config) (brainBiteHistoryStore, error) {
		return fakeStore, nil
	}
	t.Cleanup(func() {
		newBrainBiteHistoryStore = func(ctx context.Context, cfg config.Config) (brainBiteHistoryStore, error) {
			return storage.New(ctx, cfg.S3Bucket, cfg.S3Prefix, cfg.Region)
		}
	})

	gen := &fakeBrainBiteGen{
		responses: []string{"Today, let us look at burrows. Burrows are tunnels that help animals hide and stay cool."},
	}
	if _, _, err := GenerateBrainBiteWithUsage(context.Background(), time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC), cfg, gen); err != nil {
		t.Fatalf("GenerateBrainBiteWithUsage: %v", err)
	}
	if gen.calls != 1 {
		t.Fatalf("expected lesson call only, got %d", gen.calls)
	}
	if !strings.Contains(gen.prompts[0], "Previous Brain Bite coverage this week") {
		t.Fatalf("expected weekly coverage in prompt, got %q", gen.prompts[0])
	}
	if !strings.Contains(gen.prompts[0], "Today's planned subtopic: Burrows") {
		t.Fatalf("expected planned subtopic in prompt, got %q", gen.prompts[0])
	}
	if !strings.Contains(gen.prompts[0], "Yesterday's planned subtopic: Nests") || !strings.Contains(gen.prompts[0], "Tomorrow's planned subtopic: Hives") {
		t.Fatalf("expected adjacent planned subtopics in prompt, got %q", gen.prompts[0])
	}
	if !strings.Contains(gen.prompts[0], "Introduced nests as animal homes") {
		t.Fatalf("expected previous coverage summary in prompt, got %q", gen.prompts[0])
	}
	if !strings.Contains(gen.prompts[0], "Do not reintroduce Jessica") {
		t.Fatalf("expected anti-reintro prompt guidance, got %q", gen.prompts[0])
	}
	if !strings.Contains(gen.prompts[0], "connect yesterday's planned subtopic to today's") {
		t.Fatalf("expected weekly arc transition guidance, got %q", gen.prompts[0])
	}
	if !strings.Contains(gen.prompts[0], "preview of tomorrow's planned subtopic") {
		t.Fatalf("expected tomorrow preview guidance, got %q", gen.prompts[0])
	}

	var saved BrainBiteHistory
	if err := json.Unmarshal(fakeStore.uploadedData, &saved); err != nil {
		t.Fatalf("parse saved history: %v", err)
	}
	if len(saved.Weeks["2026-W17"].Coverage) != 2 {
		t.Fatalf("expected appended coverage, got %#v", saved.Weeks["2026-W17"].Coverage)
	}
}

func TestBrainBiteWeekStartUsesMonday(t *testing.T) {
	date := time.Date(2026, 4, 26, 15, 30, 0, 0, time.UTC)
	if got := brainBiteWeekKey(date); got != "2026-W17" {
		t.Fatalf("week key = %q, want 2026-W17", got)
	}
	if got := brainBiteWeekStart(date).Format("2006-01-02 15:04"); got != "2026-04-20 00:00" {
		t.Fatalf("week start = %q, want 2026-04-20 00:00", got)
	}
}

func TestLoadBrainBiteHistoryFromS3StartsEmptyWhenMissing(t *testing.T) {
	cfg := config.Default()
	cfg.S3Bucket = "brain-bite-bucket"
	fakeStore := &fakeTopicHistoryStore{downloadErr: &types.NoSuchKey{}}
	newBrainBiteHistoryStore = func(ctx context.Context, cfg config.Config) (brainBiteHistoryStore, error) {
		return fakeStore, nil
	}
	t.Cleanup(resetBrainBiteHistoryStore)

	history, err := loadBrainBiteHistory(context.Background(), cfg)
	if err != nil {
		t.Fatalf("loadBrainBiteHistory: %v", err)
	}
	if len(history.Weeks) != 0 {
		t.Fatalf("expected empty history, got %#v", history.Weeks)
	}
}

func TestLoadBrainBiteHistoryFromS3ReturnsDownloadError(t *testing.T) {
	cfg := config.Default()
	cfg.S3Bucket = "brain-bite-bucket"
	fakeStore := &fakeTopicHistoryStore{downloadErr: errors.New("S3 unavailable")}
	newBrainBiteHistoryStore = func(ctx context.Context, cfg config.Config) (brainBiteHistoryStore, error) {
		return fakeStore, nil
	}
	t.Cleanup(resetBrainBiteHistoryStore)

	_, err := loadBrainBiteHistory(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "S3 unavailable") {
		t.Fatalf("expected download error, got %v", err)
	}
}

func resetBrainBiteHistoryStore() {
	newBrainBiteHistoryStore = func(ctx context.Context, cfg config.Config) (brainBiteHistoryStore, error) {
		return storage.New(ctx, cfg.S3Bucket, cfg.S3Prefix, cfg.Region)
	}
}
