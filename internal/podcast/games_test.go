package podcast

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadGameRules(t *testing.T) {
	dir := t.TempDir()
	gameRulesDir = dir
	t.Cleanup(func() {
		gameRulesDir = filepath.Join("internal", "podcast", "games")
	})
	t.Setenv("YODEX_GAME_RULES_DIR", dir)

	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("Rule A"), 0o644); err != nil {
		t.Fatalf("write rules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte("Rule B"), 0o644); err != nil {
		t.Fatalf("write rules: %v", err)
	}

	games, err := LoadGameRules()
	if err != nil {
		t.Fatalf("LoadGameRules: %v", err)
	}
	if len(games) != 2 {
		t.Fatalf("expected 2 games, got %d", len(games))
	}
}

func TestChooseGameDeterministic(t *testing.T) {
	games := []GameRules{
		{Name: "a", Rules: "a"},
		{Name: "b", Rules: "b"},
		{Name: "c", Rules: "c"},
		{Name: "fact-or-fib", Rules: "fact"},
	}
	date := time.Date(2026, 1, 19, 0, 0, 0, 0, time.UTC) // Monday
	first, err := ChooseGame(date, games)
	if err != nil {
		t.Fatalf("ChooseGame: %v", err)
	}
	second, err := ChooseGame(date, games)
	if err != nil {
		t.Fatalf("ChooseGame: %v", err)
	}
	if first.Name != second.Name {
		t.Fatalf("expected deterministic selection, got %q and %q", first.Name, second.Name)
	}
	if first.Name != games[int(date.Weekday())%len(games)].Name {
		t.Fatalf("unexpected weekday selection: %q", first.Name)
	}
}

func TestChooseGameSundayFactOrFib(t *testing.T) {
	games := []GameRules{
		{Name: "a", Rules: "a"},
		{Name: "fact-or-fib", Rules: "fact"},
		{Name: "c", Rules: "c"},
	}
	date := time.Date(2026, 1, 18, 0, 0, 0, 0, time.UTC) // Sunday
	game, err := ChooseGame(date, games)
	if err != nil {
		t.Fatalf("ChooseGame: %v", err)
	}
	if game.Name != "fact-or-fib" {
		t.Fatalf("expected fact-or-fib on Sunday, got %q", game.Name)
	}
}

func TestChooseGameTuesdayBuildItBrainstorm(t *testing.T) {
	games := []GameRules{
		{Name: "build-it-brainstorm", Rules: "build"},
		{Name: "fact-or-fib", Rules: "fact"},
		{Name: "would-you-rather", Rules: "rather"},
	}
	date := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC) // Tuesday
	game, err := ChooseGame(date, games)
	if err != nil {
		t.Fatalf("ChooseGame: %v", err)
	}
	if game.Name != "build-it-brainstorm" {
		t.Fatalf("expected build-it-brainstorm on Tuesday, got %q", game.Name)
	}
}

func TestBuildGamePrompt(t *testing.T) {
	date := time.Date(2026, 1, 19, 0, 0, 0, 0, time.UTC) // Monday
	system, user, err := BuildGamePrompt("Space", date, GameRules{Name: "mystery", Rules: "Rule"})
	if err != nil {
		t.Fatalf("BuildGamePrompt: %v", err)
	}
	if system == "" || user == "" {
		t.Fatalf("expected prompts to be set")
	}
	if !containsAll(user, []string{"Weekday: Monday", "Topic: Space", "Game: mystery", "Then give a short, friendly summary", "Game rules:\nRule"}) {
		t.Fatalf("missing rules in prompt: %q", user)
	}
}

func containsAll(s string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
