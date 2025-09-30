package main

import (
    "errors"
    "flag"
    "log/slog"
    "os"

    cfgpkg "yodex/internal/config"
)

// yodex script
func cmdScript(args []string) error {
    var cf commonFlags
    var topic stringFlag
    var overwrite boolFlag

    fs := flag.NewFlagSet("script", flag.ContinueOnError)
    fs.SetOutput(os.Stderr)
    addCommonFlags(fs, &cf)
    fs.Var(&topic, "topic", "Explicit topic (overrides config and generation)")
    fs.Var(&overwrite, "overwrite", "Allow overwriting existing outputs")

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
    // Load and merge configuration
    fileCfg, err := cfgpkg.LoadFile(cf.config)
    if err != nil { return err }
    envOv, apiKey := cfgpkg.FromEnv()
    var flagOv cfgpkg.Overrides
    if topic.set { flagOv.Topic = &topic.v }
    if overwrite.set { flagOv.Overwrite = &overwrite.v }
    cfg := cfgpkg.Merge(fileCfg, envOv, flagOv, apiKey)

    if err := cfgpkg.ValidateForScript(cfg); err != nil { return err }
    slog.Info("script (stub)", "date", date.Format("2006-01-02"), "topic", cfg.Topic, "voice", cfg.Voice, "overwrite", cfg.Overwrite)
    // Implementation added in later tasks.
    return nil
}
