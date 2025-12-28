package main

import (
	"errors"
	"flag"
	"log/slog"
	"os"

	cfgpkg "yodex/internal/config"
)

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
	slog.Info("audio (stub)", "date", date.Format("2006-01-02"), "voice", cfg.Voice, "ttsModel", cfg.TTSModel)
	// Implementation added in later tasks.
	return nil
}
