package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/driif/go-vibe-starter/internal/api/router"
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
		if err := runServer(); err != nil {
			slog.Error("Server stopped", "error", err)
			os.Exit(1)
		}
	},
}

// init adds the server command to the root command.
func init() {
	rootCmd.AddCommand(serverCmd)
}

// runServer boots the server and blocks until a shutdown signal or a fatal
// listener error. It returns instead of exiting so every deferred cleanup runs.
func runServer() error {
	cfg := config.DefaultServiceConfigFromEnv()
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(logHandler))

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	slog.Info(
		"Starting server",
		"environment", cfg.Environment,
		"port", cfg.Server.ListenAddr,
		"auth_provider", cfg.Auth.Provider,
	)

	s, err := server.NewWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	initCtx, cancelInit := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelInit()
	if err := s.InitDB(initCtx); err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	if err := s.Initialize(); err != nil {
		return fmt.Errorf("initialize server: %w", err)
	}
	router.RegisterHandlersV1(s)

	serveErr := make(chan error, 1)
	go func() {
		if err := s.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("start server: %w", err)
		}
		slog.Info("Server closed")
		return nil
	case <-quit:
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	if err := s.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	return nil
}
