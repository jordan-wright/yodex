package podcast

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"yodex/internal/ai"
	"yodex/internal/config"
)

const wildFactSystemPrompt = "You write surprising, accurate, child-friendly facts about the natural world, space, science, music, and history for elementary-age children."

var wildFactCategories = []string{"nature", "space", "science", "music", "history"}

// GenerateWildFactWithUsage returns a short topic-independent Wild Fact and records it in history.
func GenerateWildFactWithUsage(ctx context.Context, date time.Time, cfg config.Config, gen TextGenerator, topic, continuity string) (string, ai.TokenUsage, error) {
	if gen == nil {
		return "", ai.TokenUsage{}, errors.New("ai client is required to generate a wild fact")
	}

	history, err := loadWildFactHistory(ctx, cfg)
	if err != nil {
		return "", ai.TokenUsage{}, err
	}
	if history.Entries == nil {
		history.Entries = map[string]WildFactEntry{}
	}

	category := wildFactCategory(date)
	prompt := buildWildFactPrompt(date, category, topic, recentWildFacts(history, 60), continuity)
	text, usage, err := generateTextMaybeWithUsage(ctx, gen, cfg.TextModel, wildFactSystemPrompt, prompt)
	if err != nil {
		return "", ai.TokenUsage{}, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ai.TokenUsage{}, errors.New("empty wild fact generated")
	}

	history.Entries[date.UTC().Format("2006-01-02")] = WildFactEntry{
		Category: category,
		Fact:     trimCoverageText(text, 320),
	}
	if err := saveWildFactHistory(ctx, cfg, history); err != nil {
		return "", ai.TokenUsage{}, err
	}
	return text, usage, nil
}

func wildFactCategory(date time.Time) string {
	if len(wildFactCategories) == 0 {
		return "science"
	}
	days := date.UTC().Unix() / int64(24*time.Hour/time.Second)
	index := int(days % int64(len(wildFactCategories)))
	if index < 0 {
		index += len(wildFactCategories)
	}
	return wildFactCategories[index]
}

func buildWildFactPrompt(date time.Time, category, topic string, previous []WildFactEntry, continuity string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Date: %s\n", date.UTC().Format("Monday, January 2, 2006"))
	fmt.Fprintf(&b, "Today's Wild Fact category: %s\n", category)
	fmt.Fprintf(&b, "Today's main podcast topic: %s\n", strings.TrimSpace(topic))
	if strings.TrimSpace(continuity) != "" {
		b.WriteString("\nContinuity anchor from the Brain Bite:\n")
		b.WriteString(strings.TrimSpace(continuity))
		b.WriteString("\n")
	}
	b.WriteString("\nWrite a short Wild Fact segment after the Brain Bite. Start with a playful, varied cue that signals a surprising side fact. State one accurate, genuinely funny or surprising fact in the requested category, then give a quick, clear background explanation of why it is true. Keep it to 3-4 short sentences. Do not add a heading, greeting, question, quiz, or pause tag. Do not mention Jessica, the podcast, the day's main topic, or the weekly Brain Bite. The fact must be unrelated to today's main topic. Avoid repeating or closely paraphrasing any previous Wild Fact below.")
	if len(previous) == 0 {
		return strings.TrimSpace(b.String())
	}
	b.WriteString("\n\nPrevious Wild Facts to avoid:\n")
	for _, entry := range previous {
		fmt.Fprintf(&b, "- %s: %s\n", strings.TrimSpace(entry.Category), strings.TrimSpace(entry.Fact))
	}
	return strings.TrimSpace(b.String())
}
