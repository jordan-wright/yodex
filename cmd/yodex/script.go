package main

import (
    "errors"
    "flag"
    "log/slog"
    "os"
)

// yodex script
func cmdScript(args []string) error {
    var cf commonFlags
    var topic string
    var overwrite bool

    fs := flag.NewFlagSet("script", flag.ContinueOnError)
    fs.SetOutput(os.Stderr)
    addCommonFlags(fs, &cf)
    fs.StringVar(&topic, "topic", "", "Explicit topic (overrides config and generation)")
    fs.BoolVar(&overwrite, "overwrite", false, "Allow overwriting existing outputs")

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
    slog.Info("script (stub)", "date", date.Format("2006-01-02"), "topic", topic, "config", cf.config, "overwrite", overwrite)
    // Implementation added in later tasks.
    return nil
}

