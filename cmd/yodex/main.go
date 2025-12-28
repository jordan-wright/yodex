package main

import (
	"fmt"
	"log/slog"
	"os"
)

var version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		printUsage()
		return 0
	}

	sub := args[0]
	switch sub {
	case "script":
		if err := cmdScript(args[1:]); err != nil {
			slog.Error("script failed", "err", err)
			return 1
		}
		return 0
	case "audio":
		if err := cmdAudio(args[1:]); err != nil {
			slog.Error("audio failed", "err", err)
			return 1
		}
		return 0
	case "publish":
		if err := cmdPublish(args[1:]); err != nil {
			slog.Error("publish failed", "err", err)
			return 1
		}
		return 0
	case "topic":
		if err := cmdTopic(args[1:]); err != nil {
			slog.Error("topic failed", "err", err)
			return 1
		}
		return 0
	case "all":
		if err := cmdAll(args[1:]); err != nil {
			slog.Error("all failed", "err", err)
			return 1
		}
		return 0
	case "version":
		fmt.Println(version)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", sub)
		printUsage()
		return 2
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `yodex %s

Usage:
  yodex <subcommand> [flags]

Subcommands:
  script   Generate Markdown script for a date
  audio    Generate MP3 audio from a script file/date
  publish  Upload MP3 to S3 and print URL
  topic    Print today's topic (or generate one)
  all      (optional) Run script -> audio -> publish
  version  Print version

Run "yodex <subcommand> -h" for flags.
`, version)
}
