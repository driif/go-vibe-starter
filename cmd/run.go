package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/spf13/cobra"

	"github.com/driif/go-vibe-starter/internal/db"
	"github.com/driif/go-vibe-starter/internal/identity"
	"github.com/driif/go-vibe-starter/internal/identity/keycloak"
	"github.com/driif/go-vibe-starter/internal/server"
	"github.com/driif/go-vibe-starter/internal/server/config"
	"github.com/driif/go-vibe-starter/internal/server/middleware"
)

func run(command *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	ctx := command.Context()
	startup, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	pool, err := db.Open(startup, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()
	if err := db.CheckSchema(startup, pool); err != nil {
		return err
	}
	verifier, err := newVerifier(ctx, cfg)
	if err != nil {
		return err
	}
	d := server.Deps{DB: pool, Verifier: verifier, CORSOrigins: cfg.CORSAllowedOrigins}
	var management http.Handler
	if cfg.ManagementAddr != "" {
		registry := prometheus.NewRegistry()
		registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
		d.Metrics = registry
		management = server.Management(registry)
	}
	handler, err := server.Handler(d)
	if err != nil {
		return err
	}
	var lc net.ListenConfig
	api, err := lc.Listen(ctx, "tcp", cfg.ListenAddr)
	if err != nil {
		return err
	}
	servers := map[*http.Server]net.Listener{server.New(cfg.ListenAddr, handler): api}
	if management != nil {
		admin, err := lc.Listen(ctx, "tcp", cfg.ManagementAddr)
		if err != nil {
			_ = api.Close()
			return err
		}
		servers[server.New(cfg.ManagementAddr, management)] = admin
	}
	slog.InfoContext(ctx, "server started", "addr", api.Addr().String(), "management_addr", cfg.ManagementAddr, "environment", cfg.Environment, "auth_provider", cfg.AuthProvider, "version", version)
	err = server.Serve(ctx, 25*time.Second, servers)
	slog.InfoContext(ctx, "server stopped")
	return err
}

func newVerifier(ctx context.Context, cfg config.Config) (middleware.Verifier, error) {
	if cfg.AuthProvider == "none" {
		slog.WarnContext(ctx, "authentication disabled: every request runs as the dev principal", "subject", "local-dev", "roles", cfg.AuthDevRoles, "app_environment", cfg.Environment)
		return identity.DevVerifier{Roles: cfg.AuthDevRoles}, nil
	}
	v := keycloak.New(cfg.KeycloakIssuerURL, cfg.KeycloakAudience)
	if err := v.Ready(ctx); err != nil {
		if errors.Is(err, keycloak.ErrIssuerMismatch) {
			return nil, fmt.Errorf("KEYCLOAK_ISSUER_URL: %w", err)
		}
		slog.WarnContext(ctx, "identity provider not reachable at startup", "error", err)
	} else {
		slog.InfoContext(ctx, "identity provider ready")
	}
	return v, nil
}
