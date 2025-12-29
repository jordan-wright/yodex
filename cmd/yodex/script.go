package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"yodex/internal/ai"
	cfgpkg "yodex/internal/config"
	"yodex/internal/paths"
	"yodex/internal/podcast"
)

const (
	retryMinWords = 650
)

type scriptClient interface {
	GenerateText(ctx context.Context, model, system, prompt string) (string, error)
	GenerateJSON(ctx context.Context, model, system, prompt, schemaName string, schema map[string]any) (string, error)
	GenerateTextWithUsage(ctx context.Context, model, system, prompt string) (string, ai.TokenUsage, error)
	GenerateJSONWithUsage(ctx context.Context, model, system, prompt, schemaName string, schema map[string]any) (string, ai.TokenUsage, error)
}

var newTextClient = func(apiKey string) (scriptClient, error) {
	return ai.New(apiKey, "")
}

type scriptMeta struct {
	Date      string `json:"date"`
	Topic     string `json:"topic"`
	Title     string `json:"title"`
	WordCount int    `json:"wordCount"`
	Model     string `json:"model"`
}

// yodex script
func cmdScript(args []string) error {
	var cf commonFlags
	var topic stringFlag
	var overwrite boolFlag

	fs := flag.NewFlagSet("script", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	addCommonFlags(fs, &cf)
	fs.Var(&topic, "topic", "Explicit topic (overrides config and generation)")
	fs.Var(&overwrite, "overwrite", "Allow overwriting existing outputs")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	setupLogger(cf.logLevel)
	date, err := resolveDate(cf.date)
	if err != nil {
		return err
	}
	// Load and merge configuration
	fileCfg, err := cfgpkg.LoadFile(cf.config)
	if err != nil {
		return err
	}
	envOv, apiKey := cfgpkg.FromEnv()
	var flagOv cfgpkg.Overrides
	if topic.set {
		flagOv.Topic = &topic.v
	}
	if overwrite.set {
		flagOv.Overwrite = &overwrite.v
	}
	cfg := cfgpkg.Merge(fileCfg, envOv, flagOv, apiKey)

	if err := cfgpkg.ValidateForScript(cfg); err != nil {
		return err
	}

	client, err := newTextClient(cfg.OpenAIAPIKey)
	if err != nil {
		return err
	}
	ctx := context.Background()

	slog.Info("script start", "date", date.Format("2006-01-02"), "model", cfg.TextModel)
	slog.Info("selecting topic")
	topicText, topicUsage, err := podcast.SelectTopicWithUsage(ctx, cfg, client)
	if err != nil {
		return err
	}
	slog.Info("topic selected", "topic", topicText)
	system, user, err := podcast.BuildScriptPrompts(topicText)
	if err != nil {
		return err
	}
	slog.Info("prompts built")

	episode, wordCount, rawJSON, usage, err := generateEpisode(ctx, client, cfg.TextModel, system, user)
	if err != nil {
		writeRawJSON(date, rawJSON, cfg.Overwrite)
		return err
	}
	usage = usage.Add(topicUsage)

	builder := paths.New("")
	if err := builder.EnsureOutDir(date); err != nil {
		return err
	}
	mdPath := builder.EpisodeMarkdown(date)
	metaPath := builder.EpisodeMeta(date)
	if err := paths.CheckOverwrite([]string{mdPath, metaPath}, cfg.Overwrite); err != nil {
		return err
	}

	script := episode.RenderMarkdown()
	if err := os.WriteFile(mdPath, []byte(script), 0o644); err != nil {
		return err
	}

	meta := scriptMeta{
		Date:      date.Format("2006-01-02"),
		Topic:     topicText,
		Title:     episode.Title,
		WordCount: wordCount,
		Model:     cfg.TextModel,
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(metaPath, metaBytes, 0o644); err != nil {
		return err
	}

	slog.Info(
		"script generated",
		"date", meta.Date,
		"topic", meta.Topic,
		"wordCount", meta.WordCount,
		"model", meta.Model,
		"inputTokens", usage.InputTokens,
		"outputTokens", usage.OutputTokens,
		"totalTokens", usage.TotalTokens,
		"cachedTokens", usage.CachedTokens,
		"reasoningTokens", usage.ReasoningTokens,
	)
	return nil
}

func generateEpisode(ctx context.Context, client scriptClient, model, system, user string) (podcast.Episode, int, string, ai.TokenUsage, error) {
	schema := podcast.EpisodeSchema()
	slog.Info("generating episode json")
	rawJSON, usage, err := client.GenerateJSONWithUsage(ctx, model, system, user, "episode_script", schema)
	if err != nil {
		return podcast.Episode{}, 0, rawJSON, ai.TokenUsage{}, err
	}
	slog.Info("parsing episode json")
	episode, err := podcast.ParseEpisodeJSON(rawJSON)
	if err != nil {
		return podcast.Episode{}, 0, rawJSON, usage, err
	}
	slog.Info("validating episode fields")
	if err := episode.Validate(); err != nil {
		return podcast.Episode{}, 0, rawJSON, usage, err
	}
	slog.Info("rendering markdown")
	markdown := episode.RenderMarkdown()
	wordCount := podcast.WordCount(markdown)
	if wordCount < retryMinWords {
		const correctionNote = "Tighten to about 800 words (±100) while keeping all fields complete."
		systemRetry := system + " " + correctionNote
		slog.Info("retrying episode json for length", "wordCount", wordCount)
		rawJSON, retryUsage, err := client.GenerateJSONWithUsage(ctx, model, systemRetry, user, "episode_script", schema)
		if err != nil {
			return podcast.Episode{}, 0, rawJSON, usage, err
		}
		usage = usage.Add(retryUsage)
		slog.Info("parsing retry episode json")
		episode, err = podcast.ParseEpisodeJSON(rawJSON)
		if err != nil {
			return podcast.Episode{}, 0, rawJSON, usage, err
		}
		slog.Info("validating retry episode fields")
		if err := episode.Validate(); err != nil {
			return podcast.Episode{}, 0, rawJSON, usage, err
		}
		slog.Info("rendering retry markdown")
		markdown = episode.RenderMarkdown()
		wordCount = podcast.WordCount(markdown)
	}
	slog.Info("running safety check", "wordCount", wordCount)
	if err := podcast.BasicSafetyCheck(markdown); err != nil {
		return podcast.Episode{}, 0, rawJSON, usage, err
	}
	if wordCount < retryMinWords {
		return podcast.Episode{}, 0, rawJSON, usage, errors.New("script length out of bounds after retry")
	}
	return episode, wordCount, rawJSON, usage, nil
}

func writeRawJSON(date time.Time, raw string, overwrite bool) {
	builder := paths.New("")
	dir := builder.OutDir(date)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Warn("failed to create output dir for raw json", "err", err, "dir", dir)
		return
	}
	rawPath := filepath.Join(dir, "episode.raw.json")
	if !overwrite {
		if _, err := os.Stat(rawPath); err == nil {
			slog.Warn("raw json exists; not overwriting", "path", rawPath)
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			slog.Warn("failed to check raw json path", "err", err, "path", rawPath)
			return
		}
	}
	if err := os.WriteFile(rawPath, []byte(raw), 0o644); err != nil {
		slog.Warn("failed to write raw json", "err", err, "path", rawPath)
		return
	}
	slog.Warn("wrote raw json for debugging", "path", rawPath)
}
