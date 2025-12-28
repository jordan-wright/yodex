package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds resolved configuration values after merging file, env, and flags.
type Config struct {
	Topic     string `json:"topic,omitempty"`
	Voice     string `json:"voice,omitempty"`
	S3Bucket  string `json:"s3Bucket,omitempty"`
	S3Prefix  string `json:"s3Prefix,omitempty"`
	Region    string `json:"region,omitempty"`
	Debug     bool   `json:"debug,omitempty"`
	Overwrite bool   `json:"overwrite,omitempty"`
	TextModel string `json:"textModel,omitempty"`
	TTSModel  string `json:"ttsModel,omitempty"`

	// Not persisted to file; sourced from env only.
	OpenAIAPIKey string `json:"-"`
}

// Overrides represents optional overrides from env or flags.
// Only non-nil pointers are applied during merge.
type Overrides struct {
	Topic     *string
	Voice     *string
	S3Bucket  *string
	S3Prefix  *string
	Region    *string
	Debug     *bool
	Overwrite *bool
	TextModel *string
	TTSModel  *string
}

func Default() Config {
	return Config{
		Voice:     "alloy",
		S3Prefix:  "yodex",
		TextModel: "gpt-4o-mini",
		TTSModel:  "gpt-4o-mini-tts",
	}
}

// LoadFile reads a JSON config. If file not found, returns defaults and no error.
func LoadFile(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config file: %w", err)
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config file: %w", err)
	}
	return cfg, nil
}

// FromEnv reads env vars and returns overrides and the OpenAI key.
func FromEnv() (Overrides, string) {
	var ov Overrides
	var apiKey string

	if v, ok := os.LookupEnv("YODEX_VOICE"); ok {
		ov.Voice = &[]string{v}[0]
	}
	if v, ok := os.LookupEnv("AWS_S3_BUCKET"); ok {
		ov.S3Bucket = &[]string{v}[0]
	}
	if v, ok := os.LookupEnv("AWS_S3_PREFIX"); ok {
		ov.S3Prefix = &[]string{v}[0]
	}
	if v, ok := os.LookupEnv("AWS_REGION"); ok {
		ov.Region = &[]string{v}[0]
	}
	if v, ok := os.LookupEnv("YODEX_DEBUG"); ok {
		if b, err := parseBool(v); err == nil {
			ov.Debug = &[]bool{b}[0]
		}
	}
	if v, ok := os.LookupEnv("YODEX_OVERWRITE"); ok {
		if b, err := parseBool(v); err == nil {
			ov.Overwrite = &[]bool{b}[0]
		}
	}
	if v, ok := os.LookupEnv("YODEX_TEXT_MODEL"); ok {
		ov.TextModel = &[]string{v}[0]
	}
	if v, ok := os.LookupEnv("YODEX_TTS_MODEL"); ok {
		ov.TTSModel = &[]string{v}[0]
	}
	apiKey = os.Getenv("OPENAI_API_KEY")
	return ov, apiKey
}

func parseBool(s string) (bool, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return false, fmt.Errorf("empty bool")
	}
	if s == "1" || s == "t" || s == "true" || s == "y" || s == "yes" || s == "on" {
		return true, nil
	}
	if s == "0" || s == "f" || s == "false" || s == "n" || s == "no" || s == "off" {
		return false, nil
	}
	// try strconv
	return strconv.ParseBool(s)
}

// Merge applies overrides in order: file -> env -> flags.
func Merge(fileCfg Config, env Overrides, flags Overrides, apiKey string) Config {
	cfg := fileCfg

	apply := func(ov Overrides) {
		if ov.Topic != nil {
			cfg.Topic = *ov.Topic
		}
		if ov.Voice != nil {
			cfg.Voice = *ov.Voice
		}
		if ov.S3Bucket != nil {
			cfg.S3Bucket = *ov.S3Bucket
		}
		if ov.S3Prefix != nil {
			cfg.S3Prefix = *ov.S3Prefix
		}
		if ov.Region != nil {
			cfg.Region = *ov.Region
		}
		if ov.Debug != nil {
			cfg.Debug = *ov.Debug
		}
		if ov.Overwrite != nil {
			cfg.Overwrite = *ov.Overwrite
		}
		if ov.TextModel != nil {
			cfg.TextModel = *ov.TextModel
		}
		if ov.TTSModel != nil {
			cfg.TTSModel = *ov.TTSModel
		}
	}

	apply(env)
	apply(flags)

	cfg.OpenAIAPIKey = apiKey
	return cfg
}

// Validation helpers
func ValidateForScript(cfg Config) error {
	if cfg.OpenAIAPIKey == "" {
		return errors.New("OPENAI_API_KEY is required for script generation")
	}
	if cfg.TextModel == "" {
		return errors.New("text model is required")
	}
	return nil
}

func ValidateForAudio(cfg Config) error {
	if cfg.OpenAIAPIKey == "" {
		return errors.New("OPENAI_API_KEY is required for audio generation")
	}
	if cfg.TTSModel == "" {
		return errors.New("tts model is required")
	}
	if cfg.Voice == "" {
		return errors.New("voice is required")
	}
	return nil
}

func ValidateForPublish(cfg Config) error {
	if cfg.S3Bucket == "" {
		return errors.New("S3 bucket is required for publish")
	}
	if cfg.Region == "" {
		return errors.New("AWS region is required for publish")
	}
	return nil
}
