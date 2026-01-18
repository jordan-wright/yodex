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
	}
	date := time.Date(2026, 1, 19, 0, 0, 0, 0, time.UTC)
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
}

func TestBuildGamePrompt(t *testing.T) {
	system, user, err := BuildGamePrompt("Space", GameRules{Name: "mystery", Rules: "Rule"})
	if err != nil {
		t.Fatalf("BuildGamePrompt: %v", err)
	}
	if system == "" || user == "" {
		t.Fatalf("expected prompts to be set")
	}
	if !containsAll(user, []string{"Topic: Space", "Game: mystery", "Rule"}) {
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
