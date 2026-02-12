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

func TestIntroPromptIncludesFriYayOnFriday(t *testing.T) {
	date := time.Date(2026, 1, 23, 0, 0, 0, 0, time.UTC) // Friday
	sections := StandardSectionSchema("Volcanoes", date)
	prompt := sections[0].Prompt
	if !strings.Contains(prompt, "Fri-YAY!") {
		t.Fatalf("expected Fri-YAY phrasing, got %q", prompt)
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

func TestIntroPromptIncludesTodayHolidayGuidance(t *testing.T) {
	date := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC) // Valentine's Day
	sections := StandardSectionSchema("Meteorites", date)
	prompt := sections[0].Prompt
	if !strings.Contains(prompt, "today is Valentine's Day") {
		t.Fatalf("expected holiday name in intro prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "If you're celebrating, I hope you have a wonderful holiday today.") {
		t.Fatalf("expected today holiday wish in intro prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "Before introducing \"Meteorites\"") {
		t.Fatalf("expected ordering guidance in intro prompt, got %q", prompt)
	}
}

func TestOutroPromptIncludesTomorrowHolidayGuidance(t *testing.T) {
	date := time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC) // Tomorrow is Valentine's Day
	sections := StandardSectionSchema("Meteorites", date)
	prompt := sections[2].Prompt
	if !strings.Contains(prompt, "tomorrow is Valentine's Day") {
		t.Fatalf("expected holiday name in outro prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "If you're celebrating, I hope you have a wonderful holiday tomorrow.") {
		t.Fatalf("expected tomorrow holiday wish in outro prompt, got %q", prompt)
	}
}

func TestPromptsDoNotMentionHolidayWhenNone(t *testing.T) {
	date := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	sections := StandardSectionSchema("Meteorites", date)
	if strings.Contains(sections[0].Prompt, "If you're celebrating") {
		t.Fatalf("did not expect holiday guidance in intro prompt, got %q", sections[0].Prompt)
	}
	if strings.Contains(sections[2].Prompt, "If you're celebrating") {
		t.Fatalf("did not expect holiday guidance in outro prompt, got %q", sections[2].Prompt)
	}
}
