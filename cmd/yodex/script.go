package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log/slog"
	"os"
	"strings"
	"time"

	"yodex/internal/ai"
	cfgpkg "yodex/internal/config"
	"yodex/internal/paths"
	"yodex/internal/podcast"
)

var newTextClient = func(apiKey string) (ai.TextClient, error) {
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
	envOv, apiKey, elevenLabsKey := cfgpkg.FromEnv()
	var flagOv cfgpkg.Overrides
	if topic.set {
		flagOv.Topic = &topic.v
	}
	if overwrite.set {
		flagOv.Overwrite = &overwrite.v
	}
	cfg := cfgpkg.Merge(fileCfg, envOv, flagOv, apiKey, elevenLabsKey)

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
	topicText, topicUsage, err := podcast.SelectTopicWithUsage(ctx, date, cfg, client)
	if err != nil {
		return err
	}
	slog.Info("topic selected", "topic", topicText)
	system, user, err := podcast.BuildScriptPrompts(topicText)
	if err != nil {
		return err
	}
	slog.Info("prompts built")

	episode, wordCount, usage, err := generateEpisode(ctx, client, cfg.TextModel, system, user, topicText)
	if err != nil {
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

func generateEpisode(ctx context.Context, client ai.TextClient, model, system, basePrompt, topic string) (podcast.Episode, int, ai.TokenUsage, error) {
	sections := podcast.StandardSectionSchema(topic)
	episodeSections := make([]podcast.EpisodeSection, 0, len(sections))
	var usage ai.TokenUsage
	var anchor string

	for i, spec := range sections {
		if i > 0 {
			spec.ContinuityContext = anchor
		}
		userPrompt := podcast.BuildSectionPrompt(basePrompt, spec)
		slog.Info("generating episode section", "sectionID", spec.SectionID)
		callStart := time.Now()
		text, callUsage, err := client.GenerateTextWithUsage(ctx, model, system, userPrompt)
		if err != nil {
			slog.Error("section call failed", "sectionID", spec.SectionID, "elapsed", time.Since(callStart).String(), "err", err)
			return podcast.Episode{}, 0, ai.TokenUsage{}, err
		}
		slog.Info("section received", "sectionID", spec.SectionID, "elapsed", time.Since(callStart).String())
		usage = usage.Add(callUsage)
		cleanText := strings.TrimSpace(text)
		episodeSections = append(episodeSections, podcast.EpisodeSection{
			SectionID: spec.SectionID,
			Text:      cleanText,
		})
		anchor = podcast.BuildContinuityAnchor(cleanText, spec.SectionID)
	}

	episode := podcast.Episode{
		Title:    topic,
		Sections: episodeSections,
	}
	slog.Info("validating episode fields")
	if err := episode.Validate(); err != nil {
		return podcast.Episode{}, 0, ai.TokenUsage{}, err
	}
	slog.Info("rendering markdown")
	markdown := episode.RenderMarkdown()
	wordCount := podcast.WordCount(markdown)
	slog.Info("running safety check", "wordCount", wordCount)
	if err := podcast.BasicSafetyCheck(markdown); err != nil {
		return podcast.Episode{}, 0, ai.TokenUsage{}, err
	}
	return episode, wordCount, usage, nil
}
