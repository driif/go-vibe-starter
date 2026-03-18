package cmd

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/driif/go-vibe-starter/internal/server"
	"github.com/driif/go-vibe-starter/internal/server/config"

	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "run",
	Short: "Starts the HTTP server",
	Long: `Starts the HTTP server

Requires configuration through ENV and
a fully migrated PostgreSQL database.`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

// init adds the server command to the root command.
func init() {
	rootCmd.AddCommand(serverCmd)
}

func runServer() {
	cfg := config.DefaultServiceConfigFromEnv()
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(logHandler))

	slog.Info(
		"Starting server",
		"environment", cfg.Environment,
		"port", cfg.Service.Port,
	)

	srv := server.NewWithConfig(cfg)
	if err := srv.Initialize(); err != nil {
		slog.Error("Failed to initialize server", "error", err)
		os.Exit(1)
	}

	go func() {
		if err := srv.Start(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				slog.Info("Server closed")
			} else {
				slog.Error("Failed to start server", "error", err)
				os.Exit(1)
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Failed to gracefully shut down server", "error", err)
		os.Exit(1)
	}
}
