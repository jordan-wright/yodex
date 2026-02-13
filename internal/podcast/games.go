package podcast

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type GameRules struct {
	Name  string
	Rules string
}

var gameRulesDir = filepath.Join("internal", "podcast", "games")

func LoadGameRules() ([]GameRules, error) {
	entries, err := os.ReadDir(resolveGameRulesDir())
	if err != nil {
		return nil, fmt.Errorf("read game rules dir: %w", err)
	}
	var games []GameRules
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		path := filepath.Join(resolveGameRulesDir(), name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read game rules file %s: %w", name, err)
		}
		rules := strings.TrimSpace(string(data))
		if rules == "" {
			continue
		}
		games = append(games, GameRules{
			Name:  strings.TrimSuffix(name, filepath.Ext(name)),
			Rules: rules,
		})
	}
	if len(games) == 0 {
		return nil, errors.New("no game rules found")
	}
	sort.Slice(games, func(i, j int) bool {
		return games[i].Name < games[j].Name
	})
	return games, nil
}

func resolveGameRulesDir() string {
	if v := strings.TrimSpace(os.Getenv("YODEX_GAME_RULES_DIR")); v != "" {
		return v
	}
	return gameRulesDir
}

func ChooseGame(date time.Time, games []GameRules) (GameRules, error) {
	if len(games) == 0 {
		return GameRules{}, errors.New("no games available")
	}
	weekday := date.UTC().Weekday()
	if name, ok := weekdayGameMap()[weekday]; ok {
		for _, game := range games {
			if game.Name == name {
				return game, nil
			}
		}
	}
	index := int(weekday) % len(games)
	return games[index], nil
}

func weekdayGameMap() map[time.Weekday]string {
	return map[time.Weekday]string{
		time.Sunday:   "fact-or-fib",
		time.Tuesday:  "build-it-brainstorm",
		time.Saturday: "build-it-brainstorm",
	}
}

const gameSystemPrompt = "You are a friendly, curious podcast host creating an audio-only daily game for kids ages 7–9.\n\n" +
	"The following game rules will be provided. Read and follow them exactly.\n\n" +
	"Your task:\n" +
	"- Produce ONE complete round of the game.\n" +
	"- Speak directly to the listener in a warm, encouraging tone.\n" +
	"- Keep language age-appropriate and imaginative.\n" +
	"- Assume this is audio-only (no visuals).\n\n" +
	"Interaction rules:\n" +
	"- For yes/no questions, insert the audio tag:\n" +
	"  [short pause]\n" +
	"- For free-form questions or when unsure, insert the audio tag:\n" +
	"  [short pause]\n" +
	"- Use [long pause] for imagination or reflection questions.\n" +
	"- Ask only one decision question per interaction beat.\n" +
	"- Do not stack multiple pauses for the same question.\n" +
	"- After the pause, continue as if the listener responded.\n" +
	"- Respond positively and inclusively without assuming their specific answer; keep affirmations generic and vary them.\n" +
	"- Ask one question at a time; if you ask multiple questions, split them into separate sentences and include a pause after each question.\n" +
	"- Always include a space before any audio tag; never attach tags directly to punctuation.\n\n" +
	"Audio tags:\n" +
	"- You MUST use [short pause] or [long pause] as instructed above.\n" +
	"- You SHOULD include other upbeat, voice-only audio tags to enrich delivery (e.g., [excited], [cheerful], [playful], [laughing], [short pause]).\n\n" +
	"If the game has a correct answer:\n" +
	"- Reveal the answer clearly.\n" +
	"- Say that you hope the listener got it right.\n" +
	"- Celebrate the fun of playing and learning, even if they didn't.\n\n" +
	"General style:\n" +
	"- Curious, upbeat, and kind.\n" +
	"- No sarcasm or negativity.\n" +
	"- Encourage thinking, imagination, and joy.\n" +
	"- Avoid mentioning rules explicitly during gameplay.\n" +
	"- End the game with a positive closing line (e.g., encouragement or fun fact).\n" +
	"- Do not say goodbye or reference the show ending; the outro handles that.\n\n" +
	"Now generate the game round using the provided rules."

func BuildGamePrompt(topic string, date time.Time, rules GameRules) (string, string, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return "", "", errors.New("topic is required")
	}
	if strings.TrimSpace(rules.Rules) == "" {
		return "", "", errors.New("game rules are required")
	}
	weekday := date.UTC().Weekday().String()
	user := fmt.Sprintf(
		"Weekday: %s\nTopic: %s\nGame: %s\n\nStart the game by saying: It's %s so you know what that means! It's time to play %s.\nThen give a short, friendly summary of how the game works that makes expectations clear.\n\nGame rules:\n%s",
		weekday,
		topic,
		rules.Name,
		weekday,
		rules.Name,
		rules.Rules,
	)
	return gameSystemPrompt, user, nil
}
