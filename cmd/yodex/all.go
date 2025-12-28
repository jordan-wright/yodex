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
    var voice stringFlag
    var overwrite boolFlag
    var bucket, prefix, region stringFlag

    fs := flag.NewFlagSet("all", flag.ContinueOnError)
    fs.SetOutput(os.Stderr)
    addCommonFlags(fs, &cf)
    fs.Var(&voice, "voice", "TTS voice")
    fs.Var(&overwrite, "overwrite", "Allow overwriting existing outputs")
    fs.Var(&bucket, "bucket", "S3 bucket name (required in prod)")
    fs.Var(&prefix, "prefix", "S3 key prefix")
    fs.Var(&region, "region", "AWS region (defaults from env)")
    if err := fs.Parse(args); err != nil {
        if errors.Is(err, flag.ErrHelp) {
            return nil
        }
        return err
    }

    // Share parsed flags to individual steps and log progress.
    setupLogger(cf.logLevel)
    slog.Info("running all steps")
    scriptArgs := []string{}
    if cf.date != "" { scriptArgs = append(scriptArgs, "--date", cf.date) }
    if cf.config != "" { scriptArgs = append(scriptArgs, "--config", cf.config) }
    if overwrite.set { scriptArgs = append(scriptArgs, "--overwrite", fmt.Sprint(overwrite.v)) }
    if err := cmdScript(scriptArgs); err != nil {
        return err
    }
    audioArgs := []string{}
    if cf.date != "" { audioArgs = append(audioArgs, "--date", cf.date) }
    if cf.config != "" { audioArgs = append(audioArgs, "--config", cf.config) }
    if voice.set { audioArgs = append(audioArgs, "--voice", voice.v) }
    if err := cmdAudio(audioArgs); err != nil {
        return err
    }
    publishArgs := []string{}
    if cf.date != "" { publishArgs = append(publishArgs, "--date", cf.date) }
    if cf.config != "" { publishArgs = append(publishArgs, "--config", cf.config) }
    if bucket.set { publishArgs = append(publishArgs, "--bucket", bucket.v) }
    if prefix.set { publishArgs = append(publishArgs, "--prefix", prefix.v) }
    if region.set { publishArgs = append(publishArgs, "--region", region.v) }
    if err := cmdPublish(publishArgs); err != nil {
        return err
    }
    return nil
}
