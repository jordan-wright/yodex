package main

import (
    "errors"
    "flag"
    "fmt"
    "log/slog"
    "os"
)

// yodex all (optional convenience)
func cmdAll(args []string) error {
    // Accept a minimal set of flags and reuse subcommands where possible.
    var cf commonFlags
    var voice string
    var overwrite bool
    var bucket, prefix, region string

    fs := flag.NewFlagSet("all", flag.ContinueOnError)
    fs.SetOutput(os.Stderr)
    addCommonFlags(fs, &cf)
    fs.StringVar(&voice, "voice", "alloy", "TTS voice")
    fs.BoolVar(&overwrite, "overwrite", false, "Allow overwriting existing outputs")
    fs.StringVar(&bucket, "bucket", "", "S3 bucket name (required in prod)")
    fs.StringVar(&prefix, "prefix", "yodex", "S3 key prefix")
    fs.StringVar(&region, "region", "", "AWS region (defaults from env)")
    if err := fs.Parse(args); err != nil {
        if errors.Is(err, flag.ErrHelp) {
            return nil
        }
        return err
    }

    // Share parsed flags to individual steps and log progress.
    setupLogger(cf.logLevel)
    slog.Info("running all steps")
    if err := cmdScript([]string{"--date", cf.date, "--config", cf.config, "--overwrite", fmt.Sprint(overwrite)}); err != nil {
        return err
    }
    if err := cmdAudio([]string{"--date", cf.date, "--config", cf.config, "--voice", voice}); err != nil {
        return err
    }
    if err := cmdPublish([]string{"--date", cf.date, "--config", cf.config, "--bucket", bucket, "--prefix", prefix, "--region", region}); err != nil {
        return err
    }
    return nil
}

