package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/driif/go-vibe-starter/internal/server/config/env"
	"github.com/driif/go-vibe-starter/pkg/dotenv"
	"github.com/driif/go-vibe-starter/pkg/tests"
)

type App struct {
	Environment string
	Server      Server
	Keycloak    Keycloak
	Database    Database
	Logger      Logger
	Pprof       Pprof
	Management  Management
}

type Server struct {
	ListenAddr                    string
	EnableCORSMiddleware          bool
	EnableLoggerMiddleware        bool
	EnableRecoverMiddleware       bool
	EnableRequestIDMiddleware     bool
	EnableTrailingSlashMiddleware bool
	EnableSecureMiddleware        bool
	EnableCacheControlMiddleware  bool
	SecureMiddleware              SecureMiddleware
}

type SecureMiddleware struct {
	XSSProtection         string
	ContentTypeNosniff    string
	XFrameOptions         string
	HSTSMaxAge            int
	HSTSExcludeSubdomains bool
	ContentSecurityPolicy string
	CSPReportOnly         bool
	HSTSPreloadEnabled    bool
	ReferrerPolicy        string
}

type Logger struct {
	Level             slog.Level
	RequestLevel      slog.Level
	LogRequestBody    bool
	LogRequestHeader  bool
	LogRequestQuery   bool
	LogResponseBody   bool
	LogResponseHeader bool
}

type Pprof struct {
	Enable                      bool
	EnableManagementKeyAuth     bool
	RuntimeMutexProfileFraction int
}

type Management struct {
	Secret string // sensitive — used for pprof key auth
}

type Keycloak struct {
	IssuerURL   string
	Audience    string
	HTTPTimeout time.Duration
	ClockSkew   time.Duration
}

func DefaultServiceConfigFromEnv() App {
	if !tests.RunningInTest() {
		dotenv.TryLoad(filepath.Join(env.GetProjectRootDir(), ".env.local"), os.Setenv)
	}
	return App{
		Environment: env.GetEnv("APP_ENVIRONMENT", "development"),
		Server: Server{
			ListenAddr:                    env.GetEnv("SERVICE_PORT", ":8080"),
			EnableCORSMiddleware:          env.GetEnvAsBool("SERVER_ENABLE_CORS_MIDDLEWARE", true),
			EnableLoggerMiddleware:        env.GetEnvAsBool("SERVER_ENABLE_LOGGER_MIDDLEWARE", true),
			EnableRecoverMiddleware:       env.GetEnvAsBool("SERVER_ENABLE_RECOVER_MIDDLEWARE", true),
			EnableRequestIDMiddleware:     env.GetEnvAsBool("SERVER_ENABLE_REQUEST_ID_MIDDLEWARE", true),
			EnableTrailingSlashMiddleware: env.GetEnvAsBool("SERVER_ENABLE_TRAILING_SLASH_MIDDLEWARE", true),
			EnableSecureMiddleware:        env.GetEnvAsBool("SERVER_ENABLE_SECURE_MIDDLEWARE", true),
			EnableCacheControlMiddleware:  env.GetEnvAsBool("SERVER_ENABLE_CACHE_CONTROL_MIDDLEWARE", true),
			SecureMiddleware: SecureMiddleware{
				XSSProtection:         env.GetEnv("SERVER_SECURE_XSS_PROTECTION", "1; mode=block"),
				ContentTypeNosniff:    env.GetEnv("SERVER_SECURE_CONTENT_TYPE_NOSNIFF", "nosniff"),
				XFrameOptions:         env.GetEnv("SERVER_SECURE_X_FRAME_OPTIONS", "SAMEORIGIN"),
				HSTSMaxAge:            env.GetEnvAsInt("SERVER_SECURE_HSTS_MAX_AGE", 0),
				HSTSExcludeSubdomains: env.GetEnvAsBool("SERVER_SECURE_HSTS_EXCLUDE_SUBDOMAINS", false),
				ContentSecurityPolicy: env.GetEnv("SERVER_SECURE_CONTENT_SECURITY_POLICY", ""),
				CSPReportOnly:         env.GetEnvAsBool("SERVER_SECURE_CSP_REPORT_ONLY", false),
				HSTSPreloadEnabled:    env.GetEnvAsBool("SERVER_SECURE_HSTS_PRELOAD_ENABLED", false),
				ReferrerPolicy:        env.GetEnv("SERVER_SECURE_REFERRER_POLICY", ""),
			},
		},
		Keycloak: Keycloak{
			IssuerURL: env.GetEnv(
				"KEYCLOAK_ISSUER_URL",
				env.GetEnv("KEYCLOAK_ISS", "http://localhost:8080/realms/myrealm"),
			),
			Audience: env.GetEnv(
				"KEYCLOAK_AUDIENCE",
				env.GetEnv("KEYCLOAK_CLIENT_ID", "api"),
			),
			HTTPTimeout: time.Second * time.Duration(env.GetEnvAsInt("KEYCLOAK_HTTP_TIMEOUT_SEC", 5)),
			ClockSkew:   time.Second * time.Duration(env.GetEnvAsInt("KEYCLOAK_CLOCK_SKEW_SEC", 30)),
		},
		Database: Database{
			Host:     env.GetEnv("PGHOST", "postgres"),
			Port:     env.GetEnvAsInt("PGPORT", 5432),
			Database: env.GetEnv("PGDATABASE", "development"),
			Username: env.GetEnv("PGUSER", "dbuser"),
			Password: env.GetEnv("PGPASSWORD", "dbpass"),
			AdditionalParams: map[string]string{
				"sslmode": env.GetEnv("PGSSLMODE", "disable"),
			},
			MaxOpenConns:    env.GetEnvAsInt("DB_MAX_OPEN_CONNS", runtime.NumCPU()*2),
			MaxIdleConns:    env.GetEnvAsInt("DB_MAX_IDLE_CONNS", 1),
			ConnMaxLifetime: time.Second * time.Duration(env.GetEnvAsInt("DB_CONN_MAX_LIFETIME_SEC", 60)),
		},
		Logger: Logger{
			Level:             parseSlogLevel(env.GetEnv("SERVER_LOGGER_LEVEL", "DEBUG"), slog.LevelDebug),
			RequestLevel:      parseSlogLevel(env.GetEnv("SERVER_LOGGER_REQUEST_LEVEL", "DEBUG"), slog.LevelDebug),
			LogRequestBody:    env.GetEnvAsBool("SERVER_LOGGER_LOG_REQUEST_BODY", false),
			LogRequestHeader:  env.GetEnvAsBool("SERVER_LOGGER_LOG_REQUEST_HEADER", false),
			LogRequestQuery:   env.GetEnvAsBool("SERVER_LOGGER_LOG_REQUEST_QUERY", false),
			LogResponseBody:   env.GetEnvAsBool("SERVER_LOGGER_LOG_RESPONSE_BODY", false),
			LogResponseHeader: env.GetEnvAsBool("SERVER_LOGGER_LOG_RESPONSE_HEADER", false),
		},
		Pprof: Pprof{
			Enable:                      env.GetEnvAsBool("SERVER_PPROF_ENABLE", false),
			EnableManagementKeyAuth:     env.GetEnvAsBool("SERVER_PPROF_ENABLE_MANAGEMENT_KEY_AUTH", true),
			RuntimeMutexProfileFraction: env.GetEnvAsInt("SERVER_PPROF_RUNTIME_MUTEX_PROFILE_FRACTION", 0),
		},
		Management: Management{
			Secret: env.GetEnv("SERVER_MANAGEMENT_SECRET", ""),
		},
	}
}

func parseSlogLevel(s string, def slog.Level) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return def
	}
	return l
}
