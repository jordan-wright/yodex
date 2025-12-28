package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"yodex/internal/ai"
	cfgpkg "yodex/internal/config"
	"yodex/internal/podcast"
)

// yodex topic
func cmdTopic(args []string) error {
	var cf commonFlags

	fs := flag.NewFlagSet("topic", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	addCommonFlags(fs, &cf)

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	setupLogger(cf.logLevel)

	fileCfg, err := cfgpkg.LoadFile(cf.config)
	if err != nil {
		return err
	}
	envOv, apiKey := cfgpkg.FromEnv()
	cfg := cfgpkg.Merge(fileCfg, envOv, cfgpkg.Overrides{}, apiKey)

	var client *ai.Client
	if cfg.Topic == "" {
		if cfg.OpenAIAPIKey == "" {
			return errors.New("OPENAI_API_KEY is required to generate a topic")
		}
		if cfg.TextModel == "" {
			return errors.New("text model is required to generate a topic")
		}
		client, err = ai.New(cfg.OpenAIAPIKey, "")
		if err != nil {
			return err
		}
	}

	topic, err := podcast.SelectTopic(context.Background(), cfg, client)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, topic)
	return nil
}
