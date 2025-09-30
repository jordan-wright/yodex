package main

import (
    "errors"
    "flag"
    "log/slog"
    "os"
)

// yodex audio
func cmdAudio(args []string) error {
    var cf commonFlags
    var voice string
    fs := flag.NewFlagSet("audio", flag.ContinueOnError)
    fs.SetOutput(os.Stderr)
    addCommonFlags(fs, &cf)
    fs.StringVar(&voice, "voice", "alloy", "TTS voice")

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
    slog.Info("audio (stub)", "date", date.Format("2006-01-02"), "config", cf.config, "voice", voice)
    // Implementation added in later tasks.
    return nil
}

