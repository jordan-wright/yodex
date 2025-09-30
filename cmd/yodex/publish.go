package main

import (
    "errors"
    "flag"
    "log/slog"
    "os"
)

// yodex publish
func cmdPublish(args []string) error {
    var cf commonFlags
    var bucket, prefix, region string
    var includeScript bool
    fs := flag.NewFlagSet("publish", flag.ContinueOnError)
    fs.SetOutput(os.Stderr)
    addCommonFlags(fs, &cf)
    fs.StringVar(&bucket, "bucket", "", "S3 bucket name (required in prod)")
    fs.StringVar(&prefix, "prefix", "yodex", "S3 key prefix")
    fs.StringVar(&region, "region", "", "AWS region (defaults from env)")
    fs.BoolVar(&includeScript, "include-script", false, "Also upload episode.md")

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
    slog.Info("publish (stub)", "date", date.Format("2006-01-02"), "config", cf.config, "bucket", bucket, "prefix", prefix, "region", region, "includeScript", includeScript)
    // Implementation added in later tasks.
    return nil
}

