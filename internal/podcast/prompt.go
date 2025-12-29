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
	b.WriteString("Return a JSON object matching the schema with fields: ")
	b.WriteString("title, intro, main, funFacts, jokes, recap, question. ")
	b.WriteString("Fun facts must be 3-4 items. Jokes must be 2-3 kid-safe items. ")
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
	found := findSectionHeadings(text)
	for _, section := range requiredSections {
		name := normalizeHeading(section)
		if !found[name] {
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

func findSectionHeadings(text string) map[string]bool {
	found := make(map[string]bool, len(requiredSections))
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			level := headingLevel(line)
			if level == 1 {
				found["title"] = true
			}
			name := normalizeHeading(line)
			if name != "" {
				found[name] = true
			}
			continue
		}
		name := normalizePlainHeader(line)
		if name != "" {
			found[name] = true
		}
	}
	return found
}

func normalizeHeading(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimLeft(line, "#")
	line = strings.TrimSpace(line)
	line = strings.TrimSuffix(line, ":")
	line = strings.TrimSpace(line)
	return strings.ToLower(line)
}

func normalizePlainHeader(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimSuffix(line, ":")
	line = strings.TrimSpace(line)
	return strings.ToLower(line)
}

func headingLevel(line string) int {
	level := 0
	for _, r := range line {
		if r != '#' {
			break
		}
		level++
	}
	return level
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
