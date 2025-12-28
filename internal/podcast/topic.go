package podcast

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"yodex/internal/config"
)

// TextGenerator is the small interface required for topic selection.
type TextGenerator interface {
	GenerateText(ctx context.Context, model, system, prompt string) (string, error)
}

const topicSystemPrompt = "You propose safe, accurate science topics for advanced 7-year-olds."

// SelectTopic returns the configured topic or proposes one via the AI client.
func SelectTopic(ctx context.Context, cfg config.Config, ai TextGenerator) (string, error) {
	if strings.TrimSpace(cfg.Topic) != "" {
		return strings.TrimSpace(cfg.Topic), nil
	}
	if ai == nil {
		return "", errors.New("ai client is required to generate a topic")
	}

	prompt := "Propose a single science topic for an advanced 7-year-old. " +
		"Focus on science, nature, or biology. " +
		"Reply with a short title only."
	text, err := ai.GenerateText(ctx, cfg.TextModel, topicSystemPrompt, prompt)
	if err != nil {
		return "", err
	}
	topic := sanitizeTopic(text)
	if topic == "" {
		return "", fmt.Errorf("empty topic generated")
	}
	return topic, nil
}

func sanitizeTopic(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	if idx := strings.IndexByte(trimmed, '\n'); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	trimmed = strings.TrimSpace(trimmed)
	trimmed = strings.TrimPrefix(trimmed, `"`)
	trimmed = strings.TrimSuffix(trimmed, `"`)
	trimmed = strings.TrimPrefix(trimmed, `'`)
	trimmed = strings.TrimSuffix(trimmed, `'`)
	trimmed = strings.TrimSpace(trimmed)
	trimmed = strings.TrimSuffix(trimmed, ".")
	return strings.TrimSpace(trimmed)
}
