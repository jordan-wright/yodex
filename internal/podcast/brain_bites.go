package podcast

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"yodex/internal/ai"
	"yodex/internal/config"
)

const brainBiteTopicSystemPrompt = "You propose safe, accurate weekly mini-lesson themes for elementary-age children."
const brainBitePlanSystemPrompt = "You plan a week of safe, accurate, connected mini-lessons for elementary-age children."
const brainBiteSystemPrompt = "You write safe, accurate, self-contained kid-friendly mini-lessons for elementary-age children."

// GenerateBrainBiteWithUsage returns today's Brain Bite and updates weekly Brain Bite history.
func GenerateBrainBiteWithUsage(ctx context.Context, date time.Time, cfg config.Config, gen TextGenerator) (string, ai.TokenUsage, error) {
	if gen == nil {
		return "", ai.TokenUsage{}, errors.New("ai client is required to generate a brain bite")
	}

	history, err := loadBrainBiteHistory(ctx, cfg)
	if err != nil {
		return "", ai.TokenUsage{}, err
	}
	if history.Weeks == nil {
		history.Weeks = map[string]BrainBiteWeek{}
	}

	weekKey := brainBiteWeekKey(date)
	week, usage, err := ensureBrainBiteWeek(ctx, date, cfg, gen, history, weekKey)
	if err != nil {
		return "", ai.TokenUsage{}, err
	}

	prompt := buildBrainBitePrompt(date, week, week.Coverage)
	text, callUsage, err := generateTextMaybeWithUsage(ctx, gen, cfg.TextModel, brainBiteSystemPrompt, prompt)
	if err != nil {
		return "", ai.TokenUsage{}, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ai.TokenUsage{}, errors.New("empty brain bite generated")
	}
	usage = usage.Add(callUsage)

	coverage := buildBrainBiteCoverage(date, text, indexedSubtopic(week.Subtopics, weekdayPlanIndex(date.Weekday())))
	week.Coverage = upsertBrainBiteCoverage(week.Coverage, coverage)
	history.Weeks[weekKey] = week
	if err := saveBrainBiteHistory(ctx, cfg, history); err != nil {
		return "", ai.TokenUsage{}, err
	}
	return text, usage, nil
}

func ensureBrainBiteWeek(ctx context.Context, date time.Time, cfg config.Config, gen TextGenerator, history BrainBiteHistory, weekKey string) (BrainBiteWeek, ai.TokenUsage, error) {
	if week, ok := history.Weeks[weekKey]; ok && strings.TrimSpace(week.Topic) != "" {
		if len(week.Subtopics) == 7 {
			return week, ai.TokenUsage{}, nil
		}
		subtopics, usage, err := generateBrainBitePlan(ctx, cfg, gen, week.Topic)
		if err != nil {
			return BrainBiteWeek{}, ai.TokenUsage{}, err
		}
		week.Subtopics = subtopics
		history.Weeks[weekKey] = week
		if err := saveBrainBiteHistory(ctx, cfg, history); err != nil {
			return BrainBiteWeek{}, ai.TokenUsage{}, err
		}
		return week, usage, nil
	}

	topic := strings.TrimSpace(cfg.BrainBiteTopic)
	var usage ai.TokenUsage
	if topic == "" {
		prompt := buildBrainBiteTopicPrompt(previousBrainBiteTopics(history))
		text, callUsage, err := generateTextMaybeWithUsage(ctx, gen, cfg.TextModel, brainBiteTopicSystemPrompt, prompt)
		if err != nil {
			return BrainBiteWeek{}, ai.TokenUsage{}, err
		}
		usage = usage.Add(callUsage)
		topic = sanitizeTopic(text)
	}
	if topic == "" {
		return BrainBiteWeek{}, ai.TokenUsage{}, errors.New("empty brain bite topic generated")
	}
	subtopics, planUsage, err := generateBrainBitePlan(ctx, cfg, gen, topic)
	if err != nil {
		return BrainBiteWeek{}, ai.TokenUsage{}, err
	}
	usage = usage.Add(planUsage)

	week := BrainBiteWeek{
		WeekStart: brainBiteWeekStart(date).Format("2006-01-02"),
		Topic:     topic,
		Subtopics: subtopics,
	}
	history.Weeks[weekKey] = week
	if err := saveBrainBiteHistory(ctx, cfg, history); err != nil {
		return BrainBiteWeek{}, ai.TokenUsage{}, err
	}
	return week, usage, nil
}

func buildBrainBiteTopicPrompt(previous []string) string {
	prompt := "Propose one broad weekly Brain Bite theme for a kid-friendly podcast. " +
		"The theme should support five to seven short, self-contained lessons across one week. " +
		"Good examples include different kinds of bugs, a tour of our solar system, how weather works, animal homes, or amazing rocks and minerals. " +
		"Keep it safe, accurate, concrete, and interesting for elementary-age children. " +
		"Reply with a short title only."
	if len(previous) == 0 {
		return prompt
	}

	var b strings.Builder
	b.WriteString(prompt)
	b.WriteString("\n\nPrevious Brain Bite weekly topics (do not repeat or closely paraphrase any topic in this list):\n")
	for _, topic := range previous {
		if strings.TrimSpace(topic) == "" {
			continue
		}
		fmt.Fprintf(&b, "- %s\n", strings.TrimSpace(topic))
	}
	return strings.TrimSpace(b.String())
}

func buildBrainBitePrompt(date time.Time, week BrainBiteWeek, coverage []BrainBiteCoverage) string {
	date = date.UTC()
	weekdayIndex := weekdayPlanIndex(date.Weekday())
	todaySubtopic := indexedSubtopic(week.Subtopics, weekdayIndex)
	yesterdaySubtopic := indexedSubtopic(week.Subtopics, weekdayIndex-1)
	tomorrowSubtopic := indexedSubtopic(week.Subtopics, weekdayIndex+1)

	var b strings.Builder
	fmt.Fprintf(&b, "Date: %s\n", date.Format("Monday, January 2, 2006"))
	fmt.Fprintf(&b, "Weekly Brain Bite topic: %s\n", strings.TrimSpace(week.Topic))
	if todaySubtopic != "" {
		fmt.Fprintf(&b, "Today's planned subtopic: %s\n", todaySubtopic)
	}
	if yesterdaySubtopic != "" {
		fmt.Fprintf(&b, "Yesterday's planned subtopic: %s\n", yesterdaySubtopic)
	}
	if tomorrowSubtopic != "" {
		fmt.Fprintf(&b, "Tomorrow's planned subtopic: %s\n", tomorrowSubtopic)
	}
	b.WriteString("\nWrite today's Brain Bite as a short self-contained lesson after the brain game. ")
	b.WriteString("Start with one warm, gentle transition sentence that acknowledges the game has just ended and smoothly changes gears. Then naturally say, \"it's time for the daily Brain Bite,\" before beginning the lesson. Keep this transition general: do not assume the listener gave a particular answer, repeat game details, or force a connection between the game and the weekly theme. Do not use a heading or label. Use the name \"Brain Bite\" only once, in that introduction. ")
	if date.Weekday() == time.Monday || len(coverage) == 0 {
		b.WriteString("This is the first Brain Bite for the week. After the transition and introduction, briefly name the weekly theme, teach today's subtopic, and give one enticing preview of a real later subtopic. ")
	} else if date.Weekday() == time.Sunday {
		b.WriteString("This is the final Brain Bite for the week. After the transition and introduction, briefly name the weekly theme and today's planned subtopic, teach the lesson, then close the journey with a one-sentence celebration of what listeners explored this week. ")
	} else {
		b.WriteString("After the transition and introduction, use one short sentence to name the weekly theme and connect yesterday's planned subtopic to today's. Teach today's planned subtopic and make the lesson understandable for a listener who missed earlier episodes. End with one short, playful preview of tomorrow's planned subtopic. ")
	}
	b.WriteString("Do not add a heading. Do not greet the audience again. Do not reintroduce Jessica or the podcast. Continue naturally from the game as the same host in the same conversation. ")
	b.WriteString("Use Jessica's warm first-person host voice. Keep it 3-5 short paragraphs. ")
	b.WriteString("If you mention yesterday or tomorrow, only refer to the planned subtopics listed above and do it casually in one short phrase. ")
	b.WriteString("Avoid repeating the previous coverage below; build on it with a new angle, example, fact, or mini-lesson.")

	if len(coverage) == 0 {
		return strings.TrimSpace(b.String())
	}
	b.WriteString("\n\nPrevious Brain Bite coverage this week:\n")
	for _, item := range coverage {
		if strings.TrimSpace(item.Summary) == "" {
			continue
		}
		fmt.Fprintf(&b, "- %s: %s", strings.TrimSpace(item.Date), strings.TrimSpace(item.Summary))
		if strings.TrimSpace(item.Title) != "" {
			fmt.Fprintf(&b, " (%s)", strings.TrimSpace(item.Title))
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func generateBrainBitePlan(ctx context.Context, cfg config.Config, gen TextGenerator, topic string) ([]string, ai.TokenUsage, error) {
	prompt := buildBrainBitePlanPrompt(topic)
	text, usage, err := generateTextMaybeWithUsage(ctx, gen, cfg.TextModel, brainBitePlanSystemPrompt, prompt)
	if err != nil {
		return nil, ai.TokenUsage{}, err
	}
	subtopics, err := parseBrainBitePlan(text)
	if err != nil {
		return nil, ai.TokenUsage{}, err
	}
	return subtopics, usage, nil
}

func buildBrainBitePlanPrompt(topic string) string {
	return fmt.Sprintf(
		"Create a seven-day Brain Bite plan for the weekly topic %q. "+
			"Return exactly seven short lesson titles, one for each day from Monday through Sunday, in order. "+
			"Each title should be specific enough to guide one short lesson, but broad enough to sound natural in a kid-friendly podcast. "+
			"Do not number the days in words inside the title. Reply with seven lines only.",
		strings.TrimSpace(topic),
	)
}

func parseBrainBitePlan(text string) ([]string, error) {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	subtopics := make([]string, 0, 7)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "-* ")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = trimPlanLinePrefix(line)
		subtopics = append(subtopics, strings.TrimSpace(line))
	}
	if len(subtopics) != 7 {
		return nil, fmt.Errorf("expected 7 brain bite subtopics, got %d", len(subtopics))
	}
	return subtopics, nil
}

func trimPlanLinePrefix(line string) string {
	if idx := strings.Index(line, ". "); idx > 0 && idx <= 2 {
		return strings.TrimSpace(line[idx+2:])
	}
	if idx := strings.Index(line, ") "); idx > 0 && idx <= 2 {
		return strings.TrimSpace(line[idx+2:])
	}
	return strings.TrimSpace(line)
}

func buildBrainBiteCoverage(date time.Time, text, plannedTitle string) BrainBiteCoverage {
	sentences := splitSentences(text)
	title := strings.TrimSpace(plannedTitle)
	if title == "" && len(sentences) > 0 {
		title = trimCoverageText(sentences[0], 80)
	}
	if title == "" {
		title = "Brain Bite"
	}

	summary := strings.TrimSpace(strings.Join(firstN(sentences, 2), " "))
	if summary == "" {
		summary = strings.TrimSpace(text)
	}
	return BrainBiteCoverage{
		Date:    date.UTC().Format("2006-01-02"),
		Title:   title,
		Summary: trimCoverageText(summary, 320),
	}
}

func firstN(values []string, n int) []string {
	if len(values) < n {
		return values
	}
	return values[:n]
}

func trimCoverageText(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= limit {
		return text
	}
	trimmed := strings.TrimSpace(text[:limit])
	return strings.TrimRight(trimmed, ".,;:") + "..."
}

func upsertBrainBiteCoverage(items []BrainBiteCoverage, item BrainBiteCoverage) []BrainBiteCoverage {
	out := make([]BrainBiteCoverage, 0, len(items)+1)
	for _, existing := range items {
		if existing.Date == item.Date {
			continue
		}
		out = append(out, existing)
	}
	return append(out, item)
}

func weekdayPlanIndex(day time.Weekday) int {
	switch day {
	case time.Monday:
		return 0
	case time.Tuesday:
		return 1
	case time.Wednesday:
		return 2
	case time.Thursday:
		return 3
	case time.Friday:
		return 4
	case time.Saturday:
		return 5
	case time.Sunday:
		return 6
	default:
		return -1
	}
}

func indexedSubtopic(subtopics []string, index int) string {
	if index < 0 || index >= len(subtopics) {
		return ""
	}
	return strings.TrimSpace(subtopics[index])
}

func brainBiteWeekKey(date time.Time) string {
	year, week := date.UTC().ISOWeek()
	return fmt.Sprintf("%04d-W%02d", year, week)
}

func brainBiteWeekStart(date time.Time) time.Time {
	date = date.UTC()
	offset := int(date.Weekday() - time.Monday)
	if offset < 0 {
		offset = 6
	}
	start := date.AddDate(0, 0, -offset)
	return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
}

func generateTextMaybeWithUsage(ctx context.Context, gen TextGenerator, model, system, prompt string) (string, ai.TokenUsage, error) {
	if withUsage, ok := gen.(TextGeneratorWithUsage); ok {
		return withUsage.GenerateTextWithUsage(ctx, model, system, prompt)
	}
	text, err := gen.GenerateText(ctx, model, system, prompt)
	return text, ai.TokenUsage{}, err
}
