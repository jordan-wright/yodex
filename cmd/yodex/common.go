package main

import (
    "flag"
    "fmt"
    "log/slog"
    "os"
    "strings"
    "time"
)

// set up slog logger according to level; defaults to info.
func setupLogger(level string) *slog.Logger {
    var lvl slog.Level
    switch strings.ToLower(level) {
    case "debug":
        lvl = slog.LevelDebug
    case "warn":
        lvl = slog.LevelWarn
    case "error":
        lvl = slog.LevelError
    default:
        lvl = slog.LevelInfo
    }
    h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
    logger := slog.New(h)
    slog.SetDefault(logger)
    return logger
}

// Common flags for date/config/log-level across subcommands
type commonFlags struct {
    date     string
    config   string
    logLevel string
}

func addCommonFlags(fs *flag.FlagSet, cf *commonFlags) {
    fs.StringVar(&cf.date, "date", "", "Date in YYYY-MM-DD (UTC); default: today")
    fs.StringVar(&cf.config, "config", "config.json", "Path to config file")
    fs.StringVar(&cf.logLevel, "log-level", "info", "Log level: debug, info, warn, error")
}

func resolveDate(in string) (time.Time, error) {
    if in == "" {
        return time.Now().UTC(), nil
    }
    t, err := time.Parse("2006-01-02", in)
    if err != nil {
        return time.Time{}, fmt.Errorf("invalid --date: %w", err)
    }
    return t, nil
}

// presence-aware flag types
type stringFlag struct {
    v   string
    set bool
}

func (f *stringFlag) String() string { return f.v }
func (f *stringFlag) Set(s string) error { f.v, f.set = s, true; return nil }

type boolFlag struct {
    v   bool
    set bool
}

func (f *boolFlag) String() string { if f.v { return "true" } ; return "false" }
func (f *boolFlag) Set(s string) error {
    s = strings.ToLower(strings.TrimSpace(s))
    if s == "1" || s == "t" || s == "true" || s == "y" || s == "yes" || s == "on" { f.v, f.set = true, true; return nil }
    if s == "0" || s == "f" || s == "false" || s == "n" || s == "no" || s == "off" { f.v, f.set = false, true; return nil }
    return fmt.Errorf("invalid bool: %q", s)
}

