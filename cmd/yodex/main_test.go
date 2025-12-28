package main

import (
	"os"
	"testing"
)

func TestHelp(t *testing.T) {
	if code := run([]string{"-h"}); code != 0 {
		t.Fatalf("expected help to return 0, got %d", code)
	}
}

func TestUnknownSubcommand(t *testing.T) {
	if code := run([]string{"unknown"}); code == 0 {
		t.Fatalf("expected non-zero for unknown subcommand")
	}
}

func TestScriptFlagParsing(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test")
	if code := run([]string{"script", "--date=2025-09-30", "--log-level=debug"}); code != 0 {
		t.Fatalf("script returned non-zero: %d", code)
	}
}

func TestAudioFlagParsing(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test")
	if code := run([]string{"audio", "--date=2025-09-30", "--voice=alloy"}); code != 0 {
		t.Fatalf("audio returned non-zero: %d", code)
	}
}

func TestPublishFlagParsing(t *testing.T) {
	if code := run([]string{"publish", "--date=2025-09-30", "--bucket=b", "--prefix=yodex", "--region=us-east-1"}); code != 0 {
		t.Fatalf("publish returned non-zero: %d", code)
	}
}

func TestTopicFlagParsing(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := tmp + "/config.json"
	if err := os.WriteFile(cfgPath, []byte(`{"topic":"Test Topic"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if code := run([]string{"topic", "--config", cfgPath}); code != 0 {
		t.Fatalf("topic returned non-zero: %d", code)
	}
}
