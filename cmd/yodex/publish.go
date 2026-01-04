package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	cfgpkg "yodex/internal/config"
	"yodex/internal/paths"
	"yodex/internal/storage"
)

const (
	mp3ContentType  = "audio/mpeg"
	textContentType = "text/markdown; charset=utf-8"
	jsonContentType = "application/json"
	cacheArchive    = "public, max-age=86400"
	cacheLatest     = "public, max-age=300"
)

type uploader interface {
	UploadFile(ctx context.Context, key, localPath, contentType, cacheControl string) error
	CopyToLatest(ctx context.Context, srcKey, filename, contentType, cacheControl string) error
	KeyForDate(t time.Time, filename string) string
}

var newUploader = func(ctx context.Context, bucket, prefix, region string) (uploader, error) {
	return storage.New(ctx, bucket, prefix, region)
}

// yodex publish
func cmdPublish(args []string) error {
	var cf commonFlags
	var bucket, prefix, region stringFlag
	var includeScript boolFlag
	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	addCommonFlags(fs, &cf)
	fs.Var(&bucket, "bucket", "S3 bucket name (required in prod)")
	fs.Var(&prefix, "prefix", "S3 key prefix")
	fs.Var(&region, "region", "AWS region (defaults from env)")
	fs.Var(&includeScript, "include-script", "Also upload episode.md")

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
	if bucket.set {
		flagOv.S3Bucket = &bucket.v
	}
	if prefix.set {
		flagOv.S3Prefix = &prefix.v
	}
	if region.set {
		flagOv.Region = &region.v
	}
	cfg := cfgpkg.Merge(fileCfg, envOv, flagOv, apiKey, elevenLabsKey)

	if err := cfgpkg.ValidateForPublish(cfg); err != nil {
		return err
	}

	up, err := newUploader(context.Background(), cfg.S3Bucket, cfg.S3Prefix, cfg.Region)
	if err != nil {
		return err
	}

	builder := paths.New("")
	mp3Path := builder.EpisodeMP3(date)
	mdPath := builder.EpisodeMarkdown(date)
	metaPath := builder.EpisodeMeta(date)

	if err := uploadAndCopy(context.Background(), up, date, "episode.mp3", mp3Path, mp3ContentType, cacheArchive, cacheLatest); err != nil {
		return err
	}
	if includeScript.v {
		if err := uploadAndCopy(context.Background(), up, date, "episode.md", mdPath, textContentType, cacheArchive, cacheLatest); err != nil {
			return err
		}
		if err := uploadAndCopy(context.Background(), up, date, "meta.json", metaPath, jsonContentType, cacheArchive, cacheLatest); err != nil {
			return err
		}
	}

	slog.Info("publish completed", "date", date.Format("2006-01-02"), "bucket", cfg.S3Bucket, "prefix", cfg.S3Prefix, "region", cfg.Region, "includeScript", includeScript.v)
	return nil
}

func uploadAndCopy(ctx context.Context, up uploader, date time.Time, filename, localPath, contentType, cacheArchive, cacheLatest string) error {
	if _, err := os.Stat(localPath); err != nil {
		return fmt.Errorf("missing local file %s: %w", localPath, err)
	}
	key := up.KeyForDate(date, filename)
	if err := up.UploadFile(ctx, key, localPath, contentType, cacheArchive); err != nil {
		return err
	}
	if err := up.CopyToLatest(ctx, key, filename, contentType, cacheLatest); err != nil {
		return err
	}
	return nil
}
