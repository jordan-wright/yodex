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

	episode, wordCount, usage, err := generateEpisode(ctx, date, client, cfg, system, user, topicText)
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
	sectionPaths := make([]string, 0, len(episode.Sections))
	for _, section := range episode.Sections {
		sectionPaths = append(sectionPaths, builder.EpisodeSectionMarkdown(date, section.SectionID))
	}
	pathsToCheck := append([]string{mdPath, metaPath}, sectionPaths...)
	if err := paths.CheckOverwrite(pathsToCheck, cfg.Overwrite); err != nil {
		return err
	}

	script := episode.RenderMarkdown()
	if err := os.WriteFile(mdPath, []byte(script), 0o644); err != nil {
		return err
	}
	for _, section := range episode.Sections {
		path := builder.EpisodeSectionMarkdown(date, section.SectionID)
		if err := os.WriteFile(path, []byte(strings.TrimSpace(section.Text)+"\n"), 0o644); err != nil {
			return err
		}
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

func generateEpisode(ctx context.Context, date time.Time, client ai.TextClient, cfg cfgpkg.Config, system, basePrompt, topic string) (podcast.Episode, int, ai.TokenUsage, error) {
	sections := podcast.StandardSectionSchema(topic, date)
	sectionSpecs := make(map[string]podcast.SectionSpec, len(sections))
	for _, spec := range sections {
		sectionSpecs[spec.SectionID] = spec
	}

	episodeSections := make([]podcast.EpisodeSection, 0, len(podcast.StandardSectionIDs()))
	var usage ai.TokenUsage
	var anchor string

	for _, sectionID := range []string{"intro", "topic"} {
		spec, ok := sectionSpecs[sectionID]
		if !ok {
			return podcast.Episode{}, 0, ai.TokenUsage{}, errors.New("missing section spec: " + sectionID)
		}
		section, callUsage, err := generateStandardSection(ctx, client, cfg.TextModel, system, basePrompt, spec, anchor)
		if err != nil {
			return podcast.Episode{}, 0, ai.TokenUsage{}, err
		}
		usage = usage.Add(callUsage)
		episodeSections = append(episodeSections, section)
		anchor = podcast.BuildContinuityAnchor(section.Text, section.SectionID)
	}

	gameContinuity := podcast.BuildTransitionCue(episodeSections[len(episodeSections)-1].Text, "topic")
	gameText, gameUsage, err := generateBrainGame(ctx, date, client, cfg.TextModel, topic, gameContinuity)
	if err != nil {
		return podcast.Episode{}, 0, ai.TokenUsage{}, err
	}
	usage = usage.Add(gameUsage)
	gameSection := podcast.EpisodeSection{
		SectionID: "game",
		Text:      gameText,
	}
	episodeSections = append(episodeSections, gameSection)
	anchor = podcast.BuildContinuityAnchor(gameSection.Text, gameSection.SectionID)

	brainBiteText, brainBiteUsage, err := generateBrainBite(ctx, date, client, cfg, anchor)
	if err != nil {
		return podcast.Episode{}, 0, ai.TokenUsage{}, err
	}
	usage = usage.Add(brainBiteUsage)
	brainBiteSection := podcast.EpisodeSection{
		SectionID: "brain-bite",
		Text:      brainBiteText,
	}
	episodeSections = append(episodeSections, brainBiteSection)
	anchor = podcast.BuildContinuityAnchor(brainBiteSection.Text, brainBiteSection.SectionID)

	wildFactText, wildFactUsage, err := generateWildFact(ctx, date, client, cfg, topic, anchor)
	if err != nil {
		return podcast.Episode{}, 0, ai.TokenUsage{}, err
	}
	usage = usage.Add(wildFactUsage)
	wildFactSection := podcast.EpisodeSection{
		SectionID: "wild-fact",
		Text:      wildFactText,
	}
	episodeSections = append(episodeSections, wildFactSection)
	anchor = podcast.BuildContinuityAnchor(wildFactSection.Text, wildFactSection.SectionID)

	outroSpec, ok := sectionSpecs["outro"]
	if !ok {
		return podcast.Episode{}, 0, ai.TokenUsage{}, errors.New("missing section spec: outro")
	}
	outroSection, outroUsage, err := generateStandardSection(ctx, client, cfg.TextModel, system, basePrompt, outroSpec, anchor)
	if err != nil {
		return podcast.Episode{}, 0, ai.TokenUsage{}, err
	}
	usage = usage.Add(outroUsage)
	episodeSections = append(episodeSections, outroSection)

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
	return episode, wordCount, usage, nil
}

func generateStandardSection(ctx context.Context, client ai.TextClient, model, system, basePrompt string, spec podcast.SectionSpec, anchor string) (podcast.EpisodeSection, ai.TokenUsage, error) {
	if strings.TrimSpace(anchor) != "" {
		spec.ContinuityContext = anchor
	}
	userPrompt := podcast.BuildSectionPrompt(basePrompt, spec)
	slog.Info("generating episode section", "sectionID", spec.SectionID)
	callStart := time.Now()
	text, usage, err := client.GenerateTextWithUsage(ctx, model, system, userPrompt)
	if err != nil {
		slog.Error("section call failed", "sectionID", spec.SectionID, "elapsed", time.Since(callStart).String(), "err", err)
		return podcast.EpisodeSection{}, ai.TokenUsage{}, err
	}
	slog.Info("section received", "sectionID", spec.SectionID, "elapsed", time.Since(callStart).String())
	return podcast.EpisodeSection{
		SectionID: spec.SectionID,
		Text:      strings.TrimSpace(text),
	}, usage, nil
}

func generateBrainGame(ctx context.Context, date time.Time, client ai.TextClient, model, topic, continuity string) (string, ai.TokenUsage, error) {
	games, err := podcast.LoadGameRules()
	if err != nil {
		return "", ai.TokenUsage{}, err
	}
	game, err := podcast.ChooseGame(date, games)
	if err != nil {
		return "", ai.TokenUsage{}, err
	}
	system, user, err := podcast.BuildGamePrompt(topic, date, game, continuity)
	if err != nil {
		return "", ai.TokenUsage{}, err
	}
	slog.Info("generating brain game", "game", game.Name)
	callStart := time.Now()
	text, usage, err := client.GenerateTextWithUsage(ctx, model, system, user)
	if err != nil {
		slog.Error("brain game call failed", "game", game.Name, "elapsed", time.Since(callStart).String(), "err", err)
		return "", ai.TokenUsage{}, err
	}
	slog.Info("brain game received", "game", game.Name, "elapsed", time.Since(callStart).String())
	return strings.TrimSpace(text), usage, nil
}

func generateBrainBite(ctx context.Context, date time.Time, client ai.TextClient, cfg cfgpkg.Config, continuity string) (string, ai.TokenUsage, error) {
	slog.Info("generating brain bite")
	callStart := time.Now()
	text, usage, err := podcast.GenerateBrainBiteWithUsage(ctx, date, cfg, client, continuity)
	if err != nil {
		slog.Error("brain bite call failed", "elapsed", time.Since(callStart).String(), "err", err)
		return "", ai.TokenUsage{}, err
	}
	slog.Info("brain bite received", "elapsed", time.Since(callStart).String())
	return strings.TrimSpace(text), usage, nil
}

func generateWildFact(ctx context.Context, date time.Time, client ai.TextClient, cfg cfgpkg.Config, topic, continuity string) (string, ai.TokenUsage, error) {
	slog.Info("generating wild fact")
	callStart := time.Now()
	text, usage, err := podcast.GenerateWildFactWithUsage(ctx, date, cfg, client, topic, continuity)
	if err != nil {
		slog.Error("wild fact call failed", "elapsed", time.Since(callStart).String(), "err", err)
		return "", ai.TokenUsage{}, err
	}
	slog.Info("wild fact received", "elapsed", time.Since(callStart).String())
	return strings.TrimSpace(text), usage, nil
}
