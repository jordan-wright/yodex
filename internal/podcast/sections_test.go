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
