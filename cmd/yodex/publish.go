package main

import (
    "errors"
    "flag"
    "log/slog"
    "os"

    cfgpkg "yodex/internal/config"
)

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
    if err != nil { return err }
    envOv, apiKey := cfgpkg.FromEnv()
    var flagOv cfgpkg.Overrides
    if bucket.set { flagOv.S3Bucket = &bucket.v }
    if prefix.set { flagOv.S3Prefix = &prefix.v }
    if region.set { flagOv.Region = &region.v }
    cfg := cfgpkg.Merge(fileCfg, envOv, flagOv, apiKey)

    if err := cfgpkg.ValidateForPublish(cfg); err != nil { return err }
    slog.Info("publish (stub)", "date", date.Format("2006-01-02"), "bucket", cfg.S3Bucket, "prefix", cfg.S3Prefix, "region", cfg.Region, "includeScript", includeScript.v)
    // Implementation added in later tasks.
    return nil
}
