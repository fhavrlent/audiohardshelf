// AudioHardShelf syncs audiobook listening progress from Audiobookshelf to Hardcover.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fhavrlent/audiohardshelf/internal/audiobookshelf"
	"github.com/fhavrlent/audiohardshelf/internal/config"
	"github.com/fhavrlent/audiohardshelf/internal/hardcover"
	"github.com/fhavrlent/audiohardshelf/internal/scheduler"
	"github.com/fhavrlent/audiohardshelf/internal/syncer"
)

// version is set at build time from the release tag.
var version = "dev"

const usage = `AudioHardShelf - sync Audiobookshelf listening progress to Hardcover

Usage:
  audiohardshelf [run]     Sync now, then keep syncing as you listen
  audiohardshelf sync      Sync once and exit
  audiohardshelf version   Print the version
  audiohardshelf help      Show this help
`

func main() {
	command := "run"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "run", "sync":
	case "version", "--version":
		fmt.Println(version)
		return
	case "help", "-h", "--help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", command, usage)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := start(ctx, command); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func start(ctx context.Context, command string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})))

	if missing := cfg.Missing(); len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	abs := audiobookshelf.New(cfg.ABSURL, cfg.ABSAPIKey, cfg.APITimeout)
	hc := hardcover.New(cfg.HardcoverAPIURL, cfg.HardcoverAPIKey, cfg.APITimeout)
	sync := syncer.New(abs, hc, slog.Default())

	if command == "sync" {
		return sync.Run(ctx)
	}

	slog.Info("AudioHardShelf starting", "version", version, "syncInterval", cfg.SyncInterval)

	sched := scheduler.New(sync.Run, slog.Default(), cfg.SyncInterval)
	go abs.WatchProgress(ctx, slog.Default(), sched.ProgressChanged)
	sched.Run(ctx)

	slog.Info("shutting down")
	return nil
}
