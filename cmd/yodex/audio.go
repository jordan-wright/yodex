package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"yodex/internal/ai"
	cfgpkg "yodex/internal/config"
	"yodex/internal/paths"
	"yodex/internal/podcast"
)

const (
	longPauseTag  = "[long pause]"
	shortPauseTag = "[short pause]"
)

var longPauseAudioPath = filepath.Join("assets", "audio", "pause6s.mp3")
var shortPauseAudioPath = filepath.Join("assets", "audio", "pause3s.mp3")

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
			if err := synthesizeWithPauses(ctx, client, cfg, string(text), outPath); err != nil {
				return err
			}
		}
		sectionMP3s := make([]string, 0, len(sectionIDs))
		for _, sectionID := range sectionIDs {
			sectionMP3s = append(sectionMP3s, builder.EpisodeSectionMP3(date, sectionID))
		}
		if err := concatMP3(mp3Path, sectionMP3s); err != nil {
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
		if err := synthesizeWithPauses(ctx, client, cfg, string(script), mp3Path); err != nil {
			return err
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
	if len(inputs) == 0 {
		return fmt.Errorf("no inputs to concatenate")
	}
	listFile, err := os.CreateTemp("", "yodex-mp3-list-*.txt")
	if err != nil {
		return err
	}
	defer func() {
		if cerr := os.Remove(listFile.Name()); cerr != nil {
			slog.Warn("failed to remove concat list file", "err", cerr, "path", listFile.Name())
		}
	}()
	for _, path := range inputs {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		absPath, err := filepath.Abs(trimmed)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(listFile, "file '%s'\n", escapeConcatPath(absPath)); err != nil {
			return err
		}
	}
	if err := listFile.Close(); err != nil {
		return err
	}
	outAbs, err := filepath.Abs(outPath)
	if err != nil {
		return err
	}
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "concat", "-safe", "0", "-i", listFile.Name(), "-c:a", "libmp3lame", outAbs)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("concat mp3: %w", err)
	}
	return nil
}

func escapeConcatPath(path string) string {
	return strings.ReplaceAll(path, "'", "'\\''")
}

var concatMP3 = concatMP3Files

func concatMP3ByCopy(outPath string, inputs []string) error {
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

func synthesizeWithPauses(ctx context.Context, client ai.TTSClient, cfg cfgpkg.Config, text, outPath string) error {
	segments := splitOnPauses(text)
	if len(segments) == 0 {
		return fmt.Errorf("no text to synthesize")
	}
	longPauseAbs, err := filepath.Abs(longPauseAudioPath)
	if err != nil {
		return err
	}
	shortPauseAbs, err := filepath.Abs(shortPauseAudioPath)
	if err != nil {
		return err
	}
	pausePaths := map[string]string{
		longPauseTag:  longPauseAbs,
		shortPauseTag: shortPauseAbs,
	}
	for _, segment := range segments {
		if segment.pauseTag == "" {
			continue
		}
		absPath, ok := pausePaths[segment.pauseTag]
		if !ok {
			return fmt.Errorf("unknown pause audio tag: %s", segment.pauseTag)
		}
		if _, err := os.Stat(absPath); err != nil {
			return fmt.Errorf("pause audio missing: %w", err)
		}
	}
	tmpPaths := make([]string, 0, len(segments))
	for i, segment := range segments {
		if strings.TrimSpace(segment.text) == "" {
			continue
		}
		tmpPath := fmt.Sprintf("%s.part.%02d.mp3", outPath, i)
		out, err := os.Create(tmpPath)
		if err != nil {
			return err
		}
		if err := client.TTS(ctx, cfg.TTSModel, cfg.Voice, segment.text, out); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
		tmpPaths = append(tmpPaths, tmpPath)
		if segment.pauseTag == longPauseTag {
			tmpPaths = append(tmpPaths, longPauseAbs)
		} else if segment.pauseTag == shortPauseTag {
			tmpPaths = append(tmpPaths, shortPauseAbs)
		}
	}
	if err := concatMP3(outPath, tmpPaths); err != nil {
		return err
	}
	for _, path := range tmpPaths {
		if path == longPauseAbs || path == shortPauseAbs {
			continue
		}
		if err := os.Remove(path); err != nil {
			slog.Warn("failed to remove temp audio", "err", err, "path", path)
		}
	}
	return nil
}

type pauseSegment struct {
	text     string
	pauseTag string
}

func splitOnPauses(text string) []pauseSegment {
	var segments []pauseSegment
	for len(text) > 0 {
		longIndex := strings.Index(text, longPauseTag)
		shortIndex := strings.Index(text, shortPauseTag)
		nextIndex, tag := nextPauseTag(longIndex, shortIndex)
		if nextIndex == -1 {
			segments = append(segments, pauseSegment{text: text})
			break
		}
		segment := pauseSegment{text: text[:nextIndex], pauseTag: tag}
		segments = append(segments, segment)
		text = text[nextIndex+len(tag):]
	}
	return segments
}

func nextPauseTag(longIndex, shortIndex int) (int, string) {
	if longIndex == -1 && shortIndex == -1 {
		return -1, ""
	}
	if longIndex == -1 {
		return shortIndex, shortPauseTag
	}
	if shortIndex == -1 {
		return longIndex, longPauseTag
	}
	if longIndex <= shortIndex {
		return longIndex, longPauseTag
	}
	return shortIndex, shortPauseTag
}
