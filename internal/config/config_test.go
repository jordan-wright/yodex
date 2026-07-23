package config

import (
	"testing"
)

func TestMergePrecedence(t *testing.T) {
	file := Default()
	file.Voice = "file-voice"
	file.S3Bucket = "file-bucket"
	file.BrainBiteTopic = "file-brain-bite"

	env := Overrides{}
	env.Voice = strPtr("env-voice")
	env.S3Bucket = strPtr("env-bucket")
	env.BrainBiteTopic = strPtr("env-brain-bite")

	flags := Overrides{}
	flags.Voice = strPtr("flag-voice")

	cfg := Merge(file, env, flags, "sk-key", "el-key")
	if cfg.Voice != "flag-voice" {
		t.Fatalf("voice precedence wrong: %s", cfg.Voice)
	}
	if cfg.S3Bucket != "env-bucket" {
		t.Fatalf("bucket precedence wrong: %s", cfg.S3Bucket)
	}
	if cfg.BrainBiteTopic != "env-brain-bite" {
		t.Fatalf("brain bite topic precedence wrong: %s", cfg.BrainBiteTopic)
	}
	if cfg.OpenAIAPIKey != "sk-key" {
		t.Fatalf("apikey not set")
	}
	if cfg.ElevenLabsAPIKey != "el-key" {
		t.Fatalf("elevenlabs key not set")
	}
}

func TestValidateScriptRequiresAPIKey(t *testing.T) {
	cfg := Default()
	if err := ValidateForScript(cfg); err == nil {
		t.Fatalf("expected error without OPENAI_API_KEY")
	}
	cfg.OpenAIAPIKey = "sk-test"
	if err := ValidateForScript(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("YODEX_VOICE", "env-voice")
	t.Setenv("YODEX_DEBUG", "1")
	t.Setenv("YODEX_BRAIN_BITE_TOPIC", "Animal Homes")
	t.Setenv("YODEX_BRAIN_BITE_HISTORY_PATH", "tmp/brain-bites.json")
	t.Setenv("YODEX_WILD_FACT_HISTORY_PATH", "tmp/wild-facts.json")
	t.Setenv("OPENAI_API_KEY", "sk-xyz")
	t.Setenv("ELEVENLABS_API_KEY", "el-123")
	ov, key, elevenLabsKey := FromEnv()
	if ov.Voice == nil || *ov.Voice != "env-voice" {
		t.Fatalf("voice not read from env")
	}
	if ov.Debug == nil || *ov.Debug != true {
		t.Fatalf("debug not parsed as true")
	}
	if ov.BrainBiteTopic == nil || *ov.BrainBiteTopic != "Animal Homes" {
		t.Fatalf("brain bite topic not read from env")
	}
	if ov.BrainBiteHistoryPath == nil || *ov.BrainBiteHistoryPath != "tmp/brain-bites.json" {
		t.Fatalf("brain bite history path not read from env")
	}
	if ov.WildFactHistoryPath == nil || *ov.WildFactHistoryPath != "tmp/wild-facts.json" {
		t.Fatalf("wild fact history path not read from env")
	}
	if key != "sk-xyz" {
		t.Fatalf("apikey not read from env")
	}
	if elevenLabsKey != "el-123" {
		t.Fatalf("elevenlabs key not read from env")
	}
}

func strPtr(s string) *string { return &s }
