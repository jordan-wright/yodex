package podcast

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"yodex/internal/ai"
	"yodex/internal/config"
)

// TextGenerator is the small interface required for topic selection.
type TextGenerator interface {
	GenerateText(ctx context.Context, model, system, prompt string) (string, error)
}

type TextGeneratorWithUsage interface {
	TextGenerator
	GenerateTextWithUsage(ctx context.Context, model, system, prompt string) (string, ai.TokenUsage, error)
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
		"Examples of topics: animals, cultural celebrations, science, astronomy, history, geography, physics, chemistry, biology, or nature. " +
		"The topic should be interesting and engaging for a 7-year-old. " +
		"The topic should be safe and appropriate for a 7-year-old. " +
		"You may focus on a specific animal, plant, planet, star, or some other specific thing to do a deep-dive, or you may focus on a general science topic. " +
		"The topic should be accurate and up to date. " +
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

// SelectTopicWithUsage returns the topic and token usage if available.
func SelectTopicWithUsage(ctx context.Context, cfg config.Config, gen TextGenerator) (string, ai.TokenUsage, error) {
	if strings.TrimSpace(cfg.Topic) != "" {
		return strings.TrimSpace(cfg.Topic), ai.TokenUsage{}, nil
	}
	if gen == nil {
		return "", ai.TokenUsage{}, errors.New("ai client is required to generate a topic")
	}

	prompt := "Propose a single science topic for an advanced 7-year-old. " +
		"Examples of topics: animals, cultural celebrations, science, astronomy, history, geography, physics, chemistry, biology, or nature. " +
		"The topic should be interesting and engaging for a 7-year-old. " +
		"The topic should be safe and appropriate for a 7-year-old. " +
		"You may focus on a specific animal, plant, planet, star, or some other specific thing to do a deep-dive, or you may focus on a general science topic. " +
		"The topic should be accurate and up to date. " +
		"Reply with a short title only."

	if withUsage, ok := gen.(TextGeneratorWithUsage); ok {
		text, usage, err := withUsage.GenerateTextWithUsage(ctx, cfg.TextModel, topicSystemPrompt, prompt)
		if err != nil {
			return "", ai.TokenUsage{}, err
		}
		topic := sanitizeTopic(text)
		if topic == "" {
			return "", ai.TokenUsage{}, fmt.Errorf("empty topic generated")
		}
		return topic, usage, nil
	}

	text, err := gen.GenerateText(ctx, cfg.TextModel, topicSystemPrompt, prompt)
	if err != nil {
		return "", ai.TokenUsage{}, err
	}
	topic := sanitizeTopic(text)
	if topic == "" {
		return "", ai.TokenUsage{}, fmt.Errorf("empty topic generated")
	}
	return topic, ai.TokenUsage{}, nil
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
