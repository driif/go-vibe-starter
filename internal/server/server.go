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

	"github.com/driif/go-vibe-starter/internal/server/config"
	srvmiddleware "github.com/driif/go-vibe-starter/internal/server/middleware"
	"github.com/driif/go-vibe-starter/pkg/keycloak"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	server        *http.Server
	Router        chi.Router
	Config        config.App
	DB            *sql.DB
	Auth          *keycloak.Verifier
	KeycloakAdmin *keycloak.AdminClient
	//Mailer *mailer.Mailer
	//Push   *push.Service
}

func NewWithConfig(config config.App) *Server {
	authVerifier, err := keycloak.New(keycloak.Config{
		IssuerURL:   config.Keycloak.IssuerURL,
		Audience:    config.Keycloak.Audience,
		HTTPTimeout: config.Keycloak.HTTPTimeout,
		ClockSkew:   config.Keycloak.ClockSkew,
	})
	if err != nil {
		panic(err)
	}

	s := &Server{
		Config: config,
		Router: nil,
		DB:     nil,
		Auth:   authVerifier,
	}

	if config.KeycloakAdmin.ClientID != "" {
		adminClient, err := keycloak.NewAdminClient(keycloak.AdminConfig{
			BaseURL:      config.KeycloakAdmin.BaseURL,
			Realm:        config.KeycloakAdmin.Realm,
			ClientID:     config.KeycloakAdmin.ClientID,
			ClientSecret: config.KeycloakAdmin.ClientSecret,
			HTTPTimeout:  config.Keycloak.HTTPTimeout,
		})
		if err != nil {
			slog.Warn("keycloak admin client not initialized", "error", err)
		} else {
			s.KeycloakAdmin = adminClient
		}
	}

	return s
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
		db.SetConnMaxLifetime(time.Duration(s.Config.Database.ConnMaxLifetime) * time.Second)
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
