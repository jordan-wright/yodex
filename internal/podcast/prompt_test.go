package podcast

import (
	"strings"
	"testing"
)

func TestBuildScriptPrompts(t *testing.T) {
	system, user, err := BuildScriptPrompts("Clouds and Rain")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if system == "" || user == "" {
		t.Fatalf("expected non-empty prompts")
	}
	for _, section := range RequiredSections() {
		if !strings.Contains(user, section) && !strings.Contains(system, section) {
			t.Fatalf("missing section mention in prompts: %s", section)
		}
	}
}

func TestValidateSections(t *testing.T) {
	text := "# Title\n## Intro\n## Main\n## Fun Facts\n## Jokes\n## Recap\n## Question\n"
	if err := ValidateSections(text); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateSections("# Title\n## Intro\n"); err == nil {
		t.Fatalf("expected missing sections error")
	}
}

func TestWordCount(t *testing.T) {
	count := WordCount("one two\nthree")
	if count != 3 {
		t.Fatalf("unexpected word count: %d", count)
	}
}

func TestBasicSafetyCheck(t *testing.T) {
	if err := BasicSafetyCheck("This is safe."); err != nil {
		t.Fatalf("unexpected safety error: %v", err)
	}
	if err := BasicSafetyCheck("Avoid a bomb."); err == nil {
		t.Fatalf("expected unsafe term error")
	}
}
