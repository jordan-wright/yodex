package podcast

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
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
	seed := dateSeed(date)
	rng := rand.New(rand.NewSource(seed))
	return games[rng.Intn(len(games))], nil
}

func dateSeed(date time.Time) int64 {
	utc := date.UTC()
	return int64(utc.Year()*10000 + int(utc.Month())*100 + utc.Day())
}

const gameSystemPrompt = "You are a friendly, curious podcast host creating an audio-only daily game for kids ages 7–9.\n\n" +
	"The following game rules will be provided. Read and follow them exactly.\n\n" +
	"Your task:\n" +
	"- Produce ONE complete round of the game.\n" +
	"- Speak directly to the listener in a warm, encouraging tone.\n" +
	"- Keep language age-appropriate and imaginative.\n" +
	"- Assume this is audio-only (no visuals).\n\n" +
	"Interaction rules:\n" +
	"- Whenever the listener is asked a question or invited to guess, insert the audio tag:\n" +
	"  [long pause]\n" +
	"- After the pause, continue as if the listener responded.\n" +
	"- Respond positively and inclusively, regardless of what the listener may have answered.\n\n" +
	"If the game has a correct answer:\n" +
	"- Reveal the answer clearly.\n" +
	"- Say that you hope the listener got it right.\n" +
	"- Celebrate the fun of playing and learning, even if they didn't.\n\n" +
	"General style:\n" +
	"- Curious, upbeat, and kind.\n" +
	"- No sarcasm or negativity.\n" +
	"- Encourage thinking, imagination, and joy.\n" +
	"- Avoid mentioning rules explicitly during gameplay.\n" +
	"- End the game with a positive closing line (e.g., encouragement or fun fact).\n\n" +
	"Now generate the game round using the provided rules."

func BuildGamePrompt(topic string, rules GameRules) (string, string, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return "", "", errors.New("topic is required")
	}
	if strings.TrimSpace(rules.Rules) == "" {
		return "", "", errors.New("game rules are required")
	}
	user := fmt.Sprintf(
		"Topic: %s\nGame: %s\n\nGame rules:\n%s",
		topic,
		rules.Name,
		rules.Rules,
	)
	return gameSystemPrompt, user, nil
}
