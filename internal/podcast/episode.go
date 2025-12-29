package podcast

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Episode is the structured output for a generated script.
type Episode struct {
	Title    string   `json:"title"`
	Intro    string   `json:"intro"`
	Main     string   `json:"main"`
	FunFacts []string `json:"funFacts"`
	Jokes    []string `json:"jokes"`
	Recap    string   `json:"recap"`
	Question string   `json:"question"`
}

func (e Episode) Validate() error {
	if strings.TrimSpace(e.Title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(e.Intro) == "" {
		return errors.New("intro is required")
	}
	if strings.TrimSpace(e.Main) == "" {
		return errors.New("main is required")
	}
	if len(e.FunFacts) < 3 || len(e.FunFacts) > 4 {
		return fmt.Errorf("fun facts must have 3-4 items, got %d", len(e.FunFacts))
	}
	if len(e.Jokes) < 2 || len(e.Jokes) > 3 {
		return fmt.Errorf("jokes must have 2-3 items, got %d", len(e.Jokes))
	}
	if strings.TrimSpace(e.Recap) == "" {
		return errors.New("recap is required")
	}
	if strings.TrimSpace(e.Question) == "" {
		return errors.New("question is required")
	}
	return nil
}

// EpisodeSchema returns the JSON schema used for structured output.
func EpisodeSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "Episode title",
			},
			"intro": map[string]any{
				"type":        "string",
				"description": "Intro section in friendly narrator voice",
			},
			"main": map[string]any{
				"type":        "string",
				"description": "Main section explaining the topic clearly",
			},
			"funFacts": map[string]any{
				"type":        "array",
				"minItems":    3,
				"maxItems":    4,
				"items":       map[string]any{"type": "string"},
				"description": "3-4 fun facts as short bullet items",
			},
			"jokes": map[string]any{
				"type":        "array",
				"minItems":    2,
				"maxItems":    3,
				"items":       map[string]any{"type": "string"},
				"description": "2-3 kid-safe jokes",
			},
			"recap": map[string]any{
				"type":        "string",
				"description": "Short recap",
			},
			"question": map[string]any{
				"type":        "string",
				"description": "Reflective question to end the episode",
			},
		},
		"required": []string{
			"title",
			"intro",
			"main",
			"funFacts",
			"jokes",
			"recap",
			"question",
		},
	}
}

// ParseEpisodeJSON parses the JSON output into an Episode.
func ParseEpisodeJSON(raw string) (Episode, error) {
	var ep Episode
	if err := json.Unmarshal([]byte(raw), &ep); err != nil {
		return Episode{}, err
	}
	return ep, nil
}

// RenderMarkdown returns a Markdown script for the episode.
func (e Episode) RenderMarkdown() string {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(strings.TrimSpace(e.Title))
	b.WriteString("\n\n## Intro\n\n")
	b.WriteString(strings.TrimSpace(e.Intro))
	b.WriteString("\n\n## Main\n\n")
	b.WriteString(strings.TrimSpace(e.Main))
	b.WriteString("\n\n## Fun Facts\n\n")
	for _, fact := range e.FunFacts {
		fact = strings.TrimSpace(fact)
		if fact == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(fact)
		b.WriteString("\n")
	}
	b.WriteString("\n## Jokes\n\n")
	for _, joke := range e.Jokes {
		joke = strings.TrimSpace(joke)
		if joke == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(joke)
		b.WriteString("\n")
	}
	b.WriteString("\n## Recap\n\n")
	b.WriteString(strings.TrimSpace(e.Recap))
	b.WriteString("\n\n## Question\n\n")
	b.WriteString(strings.TrimSpace(e.Question))
	b.WriteString("\n")
	return b.String()
}
