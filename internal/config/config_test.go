package config

import (
	"testing"
)

func TestMergePrecedence(t *testing.T) {
	file := Default()
	file.Voice = "file-voice"
	file.S3Bucket = "file-bucket"

	env := Overrides{}
	env.Voice = strPtr("env-voice")
	env.S3Bucket = strPtr("env-bucket")

	flags := Overrides{}
	flags.Voice = strPtr("flag-voice")

	cfg := Merge(file, env, flags, "sk-key")
	if cfg.Voice != "flag-voice" {
		t.Fatalf("voice precedence wrong: %s", cfg.Voice)
	}
	if cfg.S3Bucket != "env-bucket" {
		t.Fatalf("bucket precedence wrong: %s", cfg.S3Bucket)
	}
	if cfg.OpenAIAPIKey != "sk-key" {
		t.Fatalf("apikey not set")
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
	t.Setenv("OPENAI_API_KEY", "sk-xyz")
	ov, key := FromEnv()
	if ov.Voice == nil || *ov.Voice != "env-voice" {
		t.Fatalf("voice not read from env")
	}
	if ov.Debug == nil || *ov.Debug != true {
		t.Fatalf("debug not parsed as true")
	}
	if key != "sk-xyz" {
		t.Fatalf("apikey not read from env")
	}
}

func strPtr(s string) *string { return &s }
