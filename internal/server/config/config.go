// Package config reads the configuration from the environment. Load checks the syntax of each
// variable; Validate checks what a running server needs.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
)

// variables is every environment variable the binary reads. parse can read no other.
var variables = []string{
	"APP_ENVIRONMENT",
	"AUTH_PROVIDER",
	"AUTH_DEV_ROLES",
	"KEYCLOAK_ISSUER_URL",
	"KEYCLOAK_AUDIENCE",
	"DATABASE_URL",
	"LISTEN_ADDR",
	"MANAGEMENT_ADDR",
	"CORS_ALLOWED_ORIGINS",
	"LOG_LEVEL",
}

// devEnvironments is the allow-list for AUTH_PROVIDER=none. Everything else is a deployed
// environment.
var devEnvironments = []string{"development", "dev", "local", "test"}

// Config is the configuration of the binary.
type Config struct {
	Environment        string
	AuthProvider       string
	AuthDevRoles       []string
	KeycloakIssuerURL  string
	KeycloakAudience   string
	DatabaseURL        string
	ListenAddr         string
	ManagementAddr     string
	CORSAllowedOrigins []string
	LogLevel           slog.Level
}

// Load reads the configuration from the process environment, then from .env.local in the working
// directory, then from the built-in defaults. An empty value counts as unset. It reports every
// malformed value in one error.
func Load() (Config, error) {
	file, err := dotenv()
	if err != nil {
		return Config{}, err
	}
	return parse(func(name string) (string, bool) {
		if v := os.Getenv(name); v != "" {
			return v, true
		}
		v, ok := file[name]
		return v, ok
	})
}

func parse(lookup func(string) (string, bool)) (Config, error) {
	values := make(map[string]string, len(variables))
	for _, name := range variables {
		if v, ok := lookup(name); ok {
			values[name] = v
		}
	}
	get := func(name, fallback string) string {
		if v := values[name]; v != "" {
			return v
		}
		return fallback
	}

	var errs []error
	cfg := Config{
		Environment:        strings.ToLower(strings.TrimSpace(get("APP_ENVIRONMENT", "production"))),
		AuthProvider:       get("AUTH_PROVIDER", ""),
		AuthDevRoles:       list(get("AUTH_DEV_ROLES", "")),
		KeycloakIssuerURL:  get("KEYCLOAK_ISSUER_URL", ""),
		KeycloakAudience:   get("KEYCLOAK_AUDIENCE", ""),
		DatabaseURL:        get("DATABASE_URL", ""),
		ListenAddr:         get("LISTEN_ADDR", ":8080"),
		ManagementAddr:     get("MANAGEMENT_ADDR", ""),
		CORSAllowedOrigins: list(get("CORS_ALLOWED_ORIGINS", "")),
	}

	if p := cfg.AuthProvider; p != "" && p != "keycloak" && p != "none" {
		errs = append(errs, fmt.Errorf(`AUTH_PROVIDER %q is not supported: use "keycloak" or "none"`, p))
	}
	if v := cfg.KeycloakIssuerURL; v != "" && !validIssuer(v) {
		errs = append(errs, fmt.Errorf("KEYCLOAK_ISSUER_URL %q is not valid: use an absolute http or https URL without query, fragment or trailing slash", v))
	}
	// the value is not printed: it holds the database password
	if v := cfg.DatabaseURL; v != "" && !validDatabaseURL(v) {
		errs = append(errs, errors.New("DATABASE_URL is not valid: use a postgres:// or postgresql:// URL"))
	}
	if !validAddr(cfg.ListenAddr) {
		errs = append(errs, fmt.Errorf("LISTEN_ADDR %q is not valid: use host:port with a port from 1 to 65535", cfg.ListenAddr))
	}
	if v := cfg.ManagementAddr; v != "" && !validAddr(v) {
		errs = append(errs, fmt.Errorf("MANAGEMENT_ADDR %q is not valid: use host:port with a port from 1 to 65535", v))
	}
	for _, origin := range cfg.CORSAllowedOrigins {
		if !validOrigin(origin) {
			errs = append(errs, fmt.Errorf("CORS_ALLOWED_ORIGINS %q is not valid: use scheme://host[:port] with http or https", origin))
		}
	}
	level := get("LOG_LEVEL", "INFO")
	switch strings.ToUpper(level) {
	case "DEBUG":
		cfg.LogLevel = slog.LevelDebug
	case "INFO":
		cfg.LogLevel = slog.LevelInfo
	case "WARN":
		cfg.LogLevel = slog.LevelWarn
	case "ERROR":
		cfg.LogLevel = slog.LevelError
	default:
		errs = append(errs, fmt.Errorf("LOG_LEVEL %q is not supported: use DEBUG, INFO, WARN or ERROR", level))
	}

	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks what the run command needs beyond syntax. It reports every violation in one
// error.
func (c Config) Validate() error {
	var errs []error
	switch c.AuthProvider {
	case "":
		errs = append(errs, errors.New(`AUTH_PROVIDER is required: set "keycloak", or "none" for local development`))
	case "none":
		if !slices.Contains(devEnvironments, c.Environment) {
			errs = append(errs, fmt.Errorf("AUTH_PROVIDER=none is refused when APP_ENVIRONMENT is %q: allowed only in %s",
				c.Environment, strings.Join(devEnvironments, ", ")))
		}
	case "keycloak":
		if c.KeycloakIssuerURL == "" {
			errs = append(errs, errors.New("KEYCLOAK_ISSUER_URL is required when AUTH_PROVIDER=keycloak"))
		}
		if c.KeycloakAudience == "" {
			errs = append(errs, errors.New("KEYCLOAK_AUDIENCE is required when AUTH_PROVIDER=keycloak"))
		}
	}
	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if c.ManagementAddr != "" && c.ManagementAddr == c.ListenAddr {
		errs = append(errs, errors.New("MANAGEMENT_ADDR must differ from LISTEN_ADDR"))
	}
	return errors.Join(errs...)
}

// list splits a comma list, trims the items and drops empty ones.
func list(s string) []string {
	var items []string
	for item := range strings.SplitSeq(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func validIssuer(s string) bool {
	u, err := url.Parse(s)
	return err == nil &&
		(u.Scheme == "http" || u.Scheme == "https") &&
		u.Hostname() != "" &&
		!strings.ContainsAny(s, "?#") &&
		!strings.HasSuffix(s, "/")
}

func validDatabaseURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql")
}

func validAddr(s string) bool {
	_, port, err := net.SplitHostPort(s)
	if err != nil {
		return false
	}
	n, err := strconv.Atoi(port)
	return err == nil && n >= 1 && n <= 65535
}

func validOrigin(s string) bool {
	u, err := url.Parse(s)
	return err == nil &&
		(u.Scheme == "http" || u.Scheme == "https") &&
		u.Hostname() != "" && !strings.Contains(u.Host, "*") &&
		s == u.Scheme+"://"+u.Host
}
