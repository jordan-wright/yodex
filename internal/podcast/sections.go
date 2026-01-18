package podcast

import (
	"fmt"
	"strings"
	"time"
)

const transitionPromptSuffix = "Continue as if you are finishing the previous thought, no headings, no resets. Do not repeat the greeting."

var standardSectionIDs = []string{
	"intro",
	"core-idea",
	"deep-dive",
	"outro",
}

// SectionSpec defines the schema for a generated episode section.
type SectionSpec struct {
	SectionID              string `json:"section_id"`
	Prompt                 string `json:"prompt"`
	ContinuityContext      string `json:"continuity_context"`
	TransitionInstructions string `json:"transition_instructions"`
}

// EpisodeSection holds generated section text.
type EpisodeSection struct {
	SectionID string `json:"section_id"`
	Text      string `json:"text"`
}

// StandardSectionSchema returns the ordered section specs for an episode.
func StandardSectionSchema(topic string, date time.Time) []SectionSpec {
	return []SectionSpec{
		{
			SectionID:              "intro",
			Prompt:                 buildIntroPrompt(topic, date),
			TransitionInstructions: transitionPromptSuffix,
		},
		{
			SectionID:              "core-idea",
			Prompt:                 fmt.Sprintf("Explain the core idea about %q in a clear, curious voice. Use a relatable analogy and keep it 2-3 short paragraphs.", topic),
			TransitionInstructions: transitionPromptSuffix,
		},
		{
			SectionID:              "deep-dive",
			Prompt:                 fmt.Sprintf("Take a deeper dive into %q, building on the previous segment. Include one surprising fact and keep it 2-3 short paragraphs.", topic),
			TransitionInstructions: transitionPromptSuffix,
		},
		{
			SectionID:              "outro",
			Prompt:                 buildOutroPrompt(topic, date),
			TransitionInstructions: transitionPromptSuffix,
		},
	}
}

func buildIntroPrompt(topic string, date time.Time) string {
	dateLabel := date.UTC().Format("Monday, January 2, 2006")
	dayPhrase := "day"
	switch date.UTC().Weekday() {
	case time.Saturday, time.Sunday:
		dayPhrase = "weekend"
	}
	return fmt.Sprintf(
		"Write a warm, friendly podcast welcome for kids. Mention today's date (%s) and say you hope everyone is having a wonderful %s. Keep it 3-5 sentences, upbeat, and welcoming. Do not dive into the topic yet; just tease that you'll explore %q soon.",
		dateLabel,
		dayPhrase,
		topic,
	)
}

func buildOutroPrompt(topic string, date time.Time) string {
	dateLabel := date.UTC().Format("Monday, January 2, 2006")
	return fmt.Sprintf(
		"Wrap up the episode about %q with a friendly recap and a thoughtful question for listeners. Mention today's date (%s) and say you hope everyone has an amazing day and learns something new. Keep it 3-5 sentences.",
		topic,
		dateLabel,
	)
}

// BuildSectionPrompt builds the user prompt for a single section.
func BuildSectionPrompt(basePrompt string, spec SectionSpec) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(basePrompt))
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	fmt.Fprintf(&b, "Section ID: %s\n", spec.SectionID)
	b.WriteString("Section prompt: ")
	b.WriteString(strings.TrimSpace(spec.Prompt))
	b.WriteString("\n")
	if strings.TrimSpace(spec.ContinuityContext) != "" {
		b.WriteString("Continuity anchor:\n")
		b.WriteString(strings.TrimSpace(spec.ContinuityContext))
		b.WriteString("\n")
	}
	if strings.TrimSpace(spec.TransitionInstructions) != "" {
		b.WriteString("Transition instructions: ")
		b.WriteString(strings.TrimSpace(spec.TransitionInstructions))
	}
	return strings.TrimSpace(b.String())
}

// BuildContinuityAnchor returns the last 3-5 sentences plus a one-line summary.
func BuildContinuityAnchor(text, sectionID string) string {
	sentences := splitSentences(text)
	if len(sentences) == 0 {
		return fmt.Sprintf("State summary: The previous section (%s) just ended its main point; flow naturally into the next section.", sectionID)
	}
	start := 0
	if len(sentences) > 5 {
		start = len(sentences) - 5
	} else if len(sentences) > 3 {
		start = len(sentences) - 3
	}
	last := strings.Join(sentences[start:], " ")
	last = strings.TrimSpace(last)
	if last != "" {
		last += "\n"
	}
	last += fmt.Sprintf("State summary: The previous section (%s) just ended its main point; flow naturally into the next section.", sectionID)
	return last
}

func splitSentences(text string) []string {
	normalized := strings.ReplaceAll(strings.TrimSpace(text), "\n", " ")
	var sentences []string
	var current strings.Builder
	for _, r := range normalized {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			fragment := strings.TrimSpace(current.String())
			if fragment != "" {
				sentences = append(sentences, fragment)
			}
			current.Reset()
		}
	}
	if tail := strings.TrimSpace(current.String()); tail != "" {
		sentences = append(sentences, tail)
	}
	return sentences
}

func sectionHeading(sectionID string) string {
	switch sectionID {
	case "intro":
		return "Intro"
	case "core-idea":
		return "Core Idea"
	case "deep-dive":
		return "Deep Dive"
	case "outro":
		return "Outro"
	default:
		return strings.Title(strings.ReplaceAll(sectionID, "-", " "))
	}
}
