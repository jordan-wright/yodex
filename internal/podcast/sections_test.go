package podcast

import (
	"strings"
	"testing"
	"time"
)

func TestIntroPromptIncludesDateAndWeekend(t *testing.T) {
	date := time.Date(2026, 1, 17, 0, 0, 0, 0, time.UTC) // Saturday
	sections := StandardSectionSchema("Volcanoes", date)
	if len(sections) == 0 {
		t.Fatalf("expected sections")
	}
	prompt := sections[0].Prompt
	if !strings.Contains(prompt, "Saturday, January 17, 2026") {
		t.Fatalf("expected date in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "weekend") {
		t.Fatalf("expected weekend phrasing, got %q", prompt)
	}
}

func TestIntroPromptIncludesWeekday(t *testing.T) {
	date := time.Date(2026, 1, 19, 0, 0, 0, 0, time.UTC) // Monday
	sections := StandardSectionSchema("Volcanoes", date)
	prompt := sections[0].Prompt
	if !strings.Contains(prompt, "Monday, January 19, 2026") {
		t.Fatalf("expected date in prompt, got %q", prompt)
	}
	if strings.Contains(prompt, "weekend") {
		t.Fatalf("did not expect weekend phrasing, got %q", prompt)
	}
}

func TestIntroPromptRequiresSingleLeadIn(t *testing.T) {
	date := time.Date(2026, 1, 19, 0, 0, 0, 0, time.UTC)
	sections := StandardSectionSchema("Volcanoes", date)
	prompt := sections[0].Prompt
	if !strings.Contains(prompt, "exactly one short sentence that introduces") {
		t.Fatalf("expected single lead-in guidance, got %q", prompt)
	}
	if !strings.Contains(prompt, "Do not add a second teaser") {
		t.Fatalf("expected no-second-teaser guidance, got %q", prompt)
	}
}

func TestTopicSectionUsesDirectTransitionInstructions(t *testing.T) {
	date := time.Date(2026, 1, 19, 0, 0, 0, 0, time.UTC)
	sections := StandardSectionSchema("Volcanoes", date)
	if len(sections) < 2 {
		t.Fatalf("expected topic section")
	}
	instructions := sections[1].TransitionInstructions
	if !strings.Contains(instructions, "Do not add another greeting, teaser, or second lead-in") {
		t.Fatalf("expected anti-teaser transition guidance, got %q", instructions)
	}
	if !strings.Contains(instructions, "Start teaching the topic in the first sentence") {
		t.Fatalf("expected immediate-topic transition guidance, got %q", instructions)
	}
}
