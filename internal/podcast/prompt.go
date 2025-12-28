package podcast

import (
	"errors"
	"fmt"
	"strings"
)

const (
	TargetWords = 800
	MinWords    = 750
	MaxWords    = 900
)

var requiredSections = []string{
	"# Title",
	"## Intro",
	"## Main",
	"## Fun Facts",
	"## Jokes",
	"## Recap",
	"## Question",
}

const systemPrompt = "You are an expert kid's science podcaster for advanced 7-year-olds. " +
	"Be engaging, positive, accurate, and safe. " +
	"Use clear explanations and relatable analogies. " +
	"Avoid scary, graphic, or unsafe content."

// BuildScriptPrompts returns the system and user prompts for script generation.
func BuildScriptPrompts(topic string) (string, string, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return "", "", errors.New("topic is required")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Create an %d-word episode on %q. ", TargetWords, topic)
	b.WriteString("Output Markdown with these exact section headings in order:\n")
	for _, section := range requiredSections {
		fmt.Fprintf(&b, "%s\n", section)
	}
	b.WriteString("Fun Facts should be 3-4 bullet points. Jokes should be 2-3 kid-safe jokes. ")
	b.WriteString("Keep it upbeat, kid-safe, accurate, and easy to follow. Avoid unsafe instructions.")
	user := b.String()
	return systemPrompt, user, nil
}

// RequiredSections returns the list of required section headers.
func RequiredSections() []string {
	sections := make([]string, 0, len(requiredSections))
	sections = append(sections, requiredSections...)
	return sections
}

// ValidateSections ensures all required section headers appear in the script.
func ValidateSections(text string) error {
	var missing []string
	for _, section := range requiredSections {
		if !strings.Contains(text, section) {
			missing = append(missing, section)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required sections: %s", strings.Join(missing, ", "))
	}
	return nil
}

// WordCount returns a basic word count for the given text.
func WordCount(text string) int {
	return len(strings.Fields(text))
}

var safetyTerms = []string{
	"suicide",
	"self-harm",
	"bomb",
	"explosive",
	"gun",
	"weapon",
	"poison",
	"cocaine",
	"heroin",
	"meth",
	"sexual",
}

// BasicSafetyCheck performs a simple lexical scan for unsafe terms.
func BasicSafetyCheck(text string) error {
	lower := strings.ToLower(text)
	for _, term := range safetyTerms {
		if strings.Contains(lower, term) {
			return fmt.Errorf("unsafe term detected: %s", term)
		}
	}
	return nil
}
