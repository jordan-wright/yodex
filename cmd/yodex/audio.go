package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"yodex/internal/ai"
	cfgpkg "yodex/internal/config"
	"yodex/internal/paths"
	"yodex/internal/podcast"
)

var newTTSClient = func(cfg cfgpkg.Config) (ai.TTSClient, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.TTSProvider))
	if provider == "" {
		provider = "openai"
	}
	switch provider {
	case "openai":
		return ai.New(cfg.OpenAIAPIKey, "")
	case "elevenlabs":
		return ai.NewElevenLabs(cfg.ElevenLabsAPIKey)
	default:
		return nil, fmt.Errorf("unsupported tts provider: %s", cfg.TTSProvider)
	}
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
	envOv, apiKey, elevenLabsKey := cfgpkg.FromEnv()
	var flagOv cfgpkg.Overrides
	if voice.set {
		flagOv.Voice = &voice.v
	}
	cfg := cfgpkg.Merge(fileCfg, envOv, flagOv, apiKey, elevenLabsKey)

	if err := cfgpkg.ValidateForAudio(cfg); err != nil {
		return err
	}

	client, err := newTTSClient(cfg)
	if err != nil {
		return err
	}
	ctx := context.Background()

	builder := paths.New("")
	mdPath := builder.EpisodeMarkdown(date)
	mp3Path := builder.EpisodeMP3(date)
	if err := builder.EnsureOutDir(date); err != nil {
		return err
	}
	sectionIDs := podcast.StandardSectionIDs()
	sectionFiles := make([]string, 0, len(sectionIDs))
	for _, sectionID := range sectionIDs {
		sectionFiles = append(sectionFiles, builder.EpisodeSectionMarkdown(date, sectionID))
	}
	useSections := allFilesExist(sectionFiles)
	var mp3Paths []string
	if useSections {
		mp3Paths = make([]string, 0, len(sectionIDs)+1)
		mp3Paths = append(mp3Paths, mp3Path)
		for _, sectionID := range sectionIDs {
			mp3Paths = append(mp3Paths, builder.EpisodeSectionMP3(date, sectionID))
		}
		if err := paths.CheckOverwrite(mp3Paths, cfg.Overwrite); err != nil {
			return err
		}
		for _, sectionID := range sectionIDs {
			sectionPath := builder.EpisodeSectionMarkdown(date, sectionID)
			text, err := os.ReadFile(sectionPath)
			if err != nil {
				return err
			}
			outPath := builder.EpisodeSectionMP3(date, sectionID)
			out, err := os.Create(outPath)
			if err != nil {
				return err
			}
			if err := client.TTS(ctx, cfg.TTSModel, cfg.Voice, string(text), out); err != nil {
				_ = out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				slog.Warn("failed to close section mp3 output", "err", err, "sectionID", sectionID)
			}
		}
		sectionMP3s := make([]string, 0, len(sectionIDs))
		for _, sectionID := range sectionIDs {
			sectionMP3s = append(sectionMP3s, builder.EpisodeSectionMP3(date, sectionID))
		}
		if err := concatMP3Files(mp3Path, sectionMP3s); err != nil {
			return err
		}
	} else {
		if err := paths.CheckOverwrite([]string{mp3Path}, cfg.Overwrite); err != nil {
			return err
		}
		script, err := os.ReadFile(mdPath)
		if err != nil {
			return err
		}
		out, err := os.Create(mp3Path)
		if err != nil {
			return err
		}
		if err := client.TTS(ctx, cfg.TTSModel, cfg.Voice, string(script), out); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			slog.Warn("failed to close mp3 output", "err", err)
		}
	}

	slog.Info(
		"audio generated",
		"date", date.Format("2006-01-02"),
		"voice", cfg.Voice,
		"ttsModel", cfg.TTSModel,
		"ttsProvider", cfg.TTSProvider,
		"path", mp3Path,
	)
	return nil
}

func allFilesExist(paths []string) bool {
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			return false
		}
	}
	return true
}

func concatMP3Files(outPath string, inputs []string) error {
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil {
			slog.Warn("failed to close mp3 output", "err", cerr)
		}
	}()
	for _, path := range inputs {
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			_ = in.Close()
			return err
		}
		if err := in.Close(); err != nil {
			return err
		}
	}
	return nil
}
