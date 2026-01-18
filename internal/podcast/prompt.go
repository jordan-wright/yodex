package podcast

import (
	"errors"
	"fmt"
	"strings"
)

var requiredSections = []string{
	"# Title",
	"## Intro",
	"## Core Idea",
	"## Deep Dive",
	"## Outro",
}

const systemPrompt = "You are an expert kid's science podcaster for advanced 7-year-olds. " +
	"Be engaging, positive, accurate, and safe. " +
	"Use clear explanations and relatable analogies. " +
	"Avoid scary, graphic, or unsafe content."

// BuildScriptPrompts returns the system and base user prompt for section generation.
func BuildScriptPrompts(topic string) (string, string, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return "", "", errors.New("topic is required")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "You are writing a kid-friendly science podcast episode for the \"Curious Kids Podcast\" hosted by Jessica, about %q. ", topic)
	b.WriteString("Each request is for one section of the episode. ")
	b.WriteString("Write in a friendly narrator voice, no headings or labels. ")
	b.WriteString("Use inflection tags generously throughout the section to add energy and texture. ")
	b.WriteString("Place tags at the start of the line or sentence where they apply (e.g., before a punchline), not at the end. ")
	b.WriteString("You can stack multiple tags when it fits (e.g., [playful][excited]). ")
	b.WriteString("Use a mix of upbeat emotional tags (e.g., [happy], [excited], [curious], [encouraging], [cheerful], [warm], [playful], [storytelling], [anticipation]) and light non-verbal tags (e.g., [laughing], [chuckles], [short pause]). ")
	b.WriteString("When you ask listeners a question that invites a response, add a [pause] tag immediately after the question. ")
	b.WriteString("Avoid negative, tired, or bored tags; only use voice-related tags (no music or sound effects). ")
	b.WriteString("Examples: \"[excited][cheerful] We have a cool mystery today!\" \"[joking][playful] Why did the comet bring a suitcase?\" \"[laughing][anticipation] Because it was going on a long trip!\" ")
	b.WriteString("Keep tags brief, natural, and kid-appropriate, and never let a tag change the meaning of the sentence. ")
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
