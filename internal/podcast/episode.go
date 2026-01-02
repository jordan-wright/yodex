package podcast

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Episode is the structured output for a generated script.
type Episode struct {
	Title    string           `json:"title"`
	Sections []EpisodeSection `json:"sections"`
}

func (e Episode) Validate() error {
	if strings.TrimSpace(e.Title) == "" {
		return errors.New("title is required")
	}
	if len(e.Sections) == 0 {
		return errors.New("sections are required")
	}
	seen := make(map[string]bool, len(e.Sections))
	for _, section := range e.Sections {
		if strings.TrimSpace(section.SectionID) == "" {
			return errors.New("section_id is required")
		}
		if strings.TrimSpace(section.Text) == "" {
			return fmt.Errorf("section %q is empty", section.SectionID)
		}
		seen[section.SectionID] = true
	}
	for _, required := range standardSectionIDs {
		if !seen[required] {
			return fmt.Errorf("missing required section: %s", required)
		}
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
			"sections": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"section_id": map[string]any{
							"type":        "string",
							"description": "Section identifier",
						},
						"text": map[string]any{
							"type":        "string",
							"description": "Section content",
						},
					},
					"required": []string{"section_id", "text"},
				},
			},
		},
		"required": []string{"title", "sections"},
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
	for _, section := range e.Sections {
		b.WriteString("\n\n## ")
		b.WriteString(sectionHeading(section.SectionID))
		b.WriteString("\n\n")
		b.WriteString(strings.TrimSpace(section.Text))
	}
	b.WriteString("\n")
	return b.String()
}
