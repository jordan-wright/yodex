package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log/slog"
	"os"

	"yodex/internal/ai"
	cfgpkg "yodex/internal/config"
	"yodex/internal/paths"
)

type ttsClient interface {
	TTS(ctx context.Context, model, voice, text string, w io.Writer) error
}

var newTTSClient = func(apiKey string) (ttsClient, error) {
	return ai.New(apiKey, "")
}

// yodex audio
func cmdAudio(args []string) error {
	var cf commonFlags
	var voice stringFlag
	fs := flag.NewFlagSet("audio", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	addCommonFlags(fs, &cf)
	fs.Var(&voice, "voice", "TTS voice")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	setupLogger(cf.logLevel)
	date, err := resolveDate(cf.date)
	if err != nil {
		return err
	}
	fileCfg, err := cfgpkg.LoadFile(cf.config)
	if err != nil {
		return err
	}
	envOv, apiKey := cfgpkg.FromEnv()
	var flagOv cfgpkg.Overrides
	if voice.set {
		flagOv.Voice = &voice.v
	}
	cfg := cfgpkg.Merge(fileCfg, envOv, flagOv, apiKey)

	if err := cfgpkg.ValidateForAudio(cfg); err != nil {
		return err
	}

	client, err := newTTSClient(cfg.OpenAIAPIKey)
	if err != nil {
		return err
	}
	ctx := context.Background()

	builder := paths.New("")
	mdPath := builder.EpisodeMarkdown(date)
	mp3Path := builder.EpisodeMP3(date)
	if err := paths.CheckOverwrite([]string{mp3Path}, cfg.Overwrite); err != nil {
		return err
	}

	script, err := os.ReadFile(mdPath)
	if err != nil {
		return err
	}
	if err := builder.EnsureOutDir(date); err != nil {
		return err
	}
	out, err := os.Create(mp3Path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil {
			slog.Warn("failed to close mp3 output", "err", cerr)
		}
	}()
	if err := client.TTS(ctx, cfg.TTSModel, cfg.Voice, string(script), out); err != nil {
		return err
	}

	slog.Info("audio generated", "date", date.Format("2006-01-02"), "voice", cfg.Voice, "ttsModel", cfg.TTSModel, "path", mp3Path)
	return nil
}
