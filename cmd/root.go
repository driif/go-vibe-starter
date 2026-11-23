// Package cmd provides the server and database commands.
package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/driif/go-vibe-starter/internal/reqlog"
	"github.com/driif/go-vibe-starter/internal/server/config"
)

var version = "dev"

// Execute runs the CLI until the command finishes or receives a signal.
func Execute() {
	setLogger(slog.LevelInfo)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	context.AfterFunc(ctx, stop)
	err := root().ExecuteContext(ctx)
	stop()
	if err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func root() *cobra.Command {
	r := &cobra.Command{Use: filepath.Base(os.Args[0]), Version: version, SilenceUsage: true, SilenceErrors: true}
	r.AddCommand(&cobra.Command{Use: "run", Short: "Serve the API", Args: cobra.NoArgs, RunE: run},
		&cobra.Command{Use: "probe", Short: "Check API readiness", Args: cobra.NoArgs, RunE: probe}, databaseCommand())
	return r
}

func setLogger(level slog.Level) {
	slog.SetDefault(slog.New(reqlog.NewHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))))
}

func loadConfig() (config.Config, error) {
	c, err := config.Load()
	if err == nil {
		setLogger(c.LogLevel)
	}
	return c, err
}
