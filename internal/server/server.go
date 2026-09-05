package server

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"runtime"
	"strings"
	"time"

	"github.com/driif/go-vibe-starter/internal/server/auth"
	"github.com/driif/go-vibe-starter/internal/server/config"
	srvmiddleware "github.com/driif/go-vibe-starter/internal/server/middleware"
	"github.com/driif/go-vibe-starter/pkg/keycloak"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Server owns the HTTP listener and the dependencies handlers reach through.
type Server struct {
	server *http.Server
	Router chi.Router
	Config config.App
	DB     *sql.DB
	// Auth is nil when Config.Auth.Provider is config.AuthProviderNone.
	Auth          *keycloak.Verifier
	KeycloakAdmin *keycloak.AdminClient
}

// NewWithConfig builds a server from config. Call config.App.Validate first:
// this constructor assumes the auth provider is one of the accepted values.
func NewWithConfig(cfg config.App) (*Server, error) {
	s := &Server{Config: cfg}

	if cfg.Auth.Provider == config.AuthProviderNone {
		slog.Warn("auth disabled — all requests run as the dev principal",
			"subject", cfg.Auth.DevSubject,
			"realm_roles", cfg.Auth.DevRoles,
		)
	} else {
		verifier, err := keycloak.New(keycloak.Config{
			IssuerURL:   cfg.Keycloak.IssuerURL,
			Audience:    cfg.Keycloak.Audience,
			HTTPTimeout: cfg.Keycloak.HTTPTimeout,
			ClockSkew:   cfg.Keycloak.ClockSkew,
		})
		if err != nil {
			return nil, fmt.Errorf("keycloak verifier: %w", err)
		}
		s.Auth = verifier
	}

	if cfg.KeycloakAdmin.ClientID != "" {
		adminClient, err := keycloak.NewAdminClient(keycloak.AdminConfig{
			BaseURL:      cfg.KeycloakAdmin.BaseURL,
			Realm:        cfg.KeycloakAdmin.Realm,
			ClientID:     cfg.KeycloakAdmin.ClientID,
			ClientSecret: cfg.KeycloakAdmin.ClientSecret,
			HTTPTimeout:  cfg.Keycloak.HTTPTimeout,
		})
		if err != nil {
			slog.Warn("keycloak admin client not initialized", "error", err)
		} else {
			s.KeycloakAdmin = adminClient
		}
	}

	return s, nil
}

// Authenticator returns the middleware that puts the caller's principal in the
// request context: token verification for keycloak, a static principal for none.
func (s *Server) Authenticator() func(http.Handler) http.Handler {
	if s.Config.Auth.Provider == config.AuthProviderNone {
		return auth.DevPrincipal(s.Config.Auth.DevSubject, s.Config.Auth.DevRoles)
	}
	return auth.Authenticate(s.Auth, auth.Options{})
}

func (s *Server) Ready() bool {
	return s.DB != nil && s.Router != nil
}

func (s *Server) InitDB(ctx context.Context) error {
	db, err := sql.Open("postgres", s.Config.Database.ConnectionString())
	if err != nil {
		return err
	}

	if s.Config.Database.MaxOpenConns > 0 {
		db.SetMaxOpenConns(s.Config.Database.MaxOpenConns)
	}
	if s.Config.Database.MaxIdleConns > 0 {
		db.SetMaxIdleConns(s.Config.Database.MaxIdleConns)
	}
	if s.Config.Database.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(s.Config.Database.ConnMaxLifetime)
	}

	if err := db.PingContext(ctx); err != nil {
		return err
	}

	s.DB = db
	return nil
}

func (s *Server) Initialize() error {
	if s.server != nil {
		return nil
	}

	addr := s.Config.Server.ListenAddr
	if addr == "" {
		return fmt.Errorf("server listen address not set")
	}
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	r := chi.NewRouter()

	if s.Config.Server.EnableTrailingSlashMiddleware {
		r.Use(chimiddleware.StripSlashes)
	} else {
		slog.Warn("trailing slash middleware disabled")
	}

	if s.Config.Server.EnableRecoverMiddleware {
		r.Use(chimiddleware.Recoverer)
	} else {
		slog.Warn("recover middleware disabled")
	}

	if s.Config.Server.EnableSecureMiddleware {
		sc := s.Config.Server.SecureMiddleware
		r.Use(srvmiddleware.Secure(srvmiddleware.SecureConfig{
			XSSProtection:         sc.XSSProtection,
			ContentTypeNosniff:    sc.ContentTypeNosniff,
			XFrameOptions:         sc.XFrameOptions,
			HSTSMaxAge:            sc.HSTSMaxAge,
			HSTSExcludeSubdomains: sc.HSTSExcludeSubdomains,
			ContentSecurityPolicy: sc.ContentSecurityPolicy,
			CSPReportOnly:         sc.CSPReportOnly,
			HSTSPreloadEnabled:    sc.HSTSPreloadEnabled,
			ReferrerPolicy:        sc.ReferrerPolicy,
		}))
	} else {
		slog.Warn("secure middleware disabled")
	}

	if s.Config.Server.EnableRequestIDMiddleware {
		r.Use(chimiddleware.RequestID)
	} else {
		slog.Warn("request ID middleware disabled")
	}

	if s.Config.Server.EnableLoggerMiddleware {
		lc := s.Config.Logger
		r.Use(srvmiddleware.LoggerWithConfig(srvmiddleware.LoggerConfig{
			Level:             lc.RequestLevel,
			LogRequestBody:    lc.LogRequestBody,
			LogRequestHeader:  lc.LogRequestHeader,
			LogRequestQuery:   lc.LogRequestQuery,
			LogResponseBody:   lc.LogResponseBody,
			LogResponseHeader: lc.LogResponseHeader,
		}))
	} else {
		slog.Warn("logger middleware disabled")
	}

	if s.Config.Server.EnableCORSMiddleware {
		r.Use(cors.AllowAll().Handler)
	} else {
		slog.Warn("CORS middleware disabled")
	}

	if s.Config.Server.EnableCacheControlMiddleware {
		r.Use(srvmiddleware.CacheControl)
	} else {
		slog.Warn("cache control middleware disabled")
	}

	if s.Config.Pprof.Enable {
		s.mountPprof(r)
	}

	s.Router = r
	s.server = &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return nil
}

func (s *Server) Start() error {
	if err := s.Initialize(); err != nil {
		return err
	}
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// mountPprof registers pprof debug routes, optionally guarded by a management secret.
func (s *Server) mountPprof(r chi.Router) {
	cfg := s.Config.Pprof

	if cfg.RuntimeMutexProfileFraction != 0 {
		runtime.SetMutexProfileFraction(cfg.RuntimeMutexProfileFraction)
		slog.Warn("pprof mutex profile fraction set", "fraction", cfg.RuntimeMutexProfileFraction)
	}

	var guard func(http.Handler) http.Handler
	if cfg.EnableManagementKeyAuth && s.Config.Management.Secret != "" {
		secret := s.Config.Management.Secret
		guard = func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("mgmt-secret") != secret {
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
			})
		}
	} else {
		guard = func(next http.Handler) http.Handler { return next }
	}

	r.With(guard).Get("/debug/pprof", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/debug/pprof/", http.StatusMovedPermanently)
	})
	r.With(guard).Get("/debug/pprof/*", func(w http.ResponseWriter, r *http.Request) {
		http.DefaultServeMux.ServeHTTP(w, r)
	})
	r.With(guard).Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	r.With(guard).Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	r.With(guard).Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	r.With(guard).Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	slog.Warn("pprof handlers available at /debug/pprof", "management_key_auth", cfg.EnableManagementKeyAuth)
}
