package config

import (
	"log/slog"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lookupIn(env map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		v, ok := env[name]
		return v, ok
	}
}

// isolate moves the test into an empty directory and unsets every variable the reader knows.
func isolate(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	for _, name := range variables {
		t.Setenv(name, "")
	}
}

func writeDotenv(t *testing.T, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(dotenvFile, []byte(content), 0o600))
}

func TestParse(t *testing.T) {
	valid := []struct {
		name string
		env  map[string]string
		want Config
	}{
		{
			name: "nothing set gives the built-in defaults",
			env:  nil,
			want: Config{Environment: "production", ListenAddr: ":8080", LogLevel: slog.LevelInfo},
		},
		{
			name: "an empty value counts as unset",
			env:  map[string]string{"APP_ENVIRONMENT": "", "LISTEN_ADDR": "", "LOG_LEVEL": "", "AUTH_PROVIDER": ""},
			want: Config{Environment: "production", ListenAddr: ":8080", LogLevel: slog.LevelInfo},
		},
		{
			name: "every variable set",
			env: map[string]string{
				"APP_ENVIRONMENT":      "development",
				"AUTH_PROVIDER":        "keycloak",
				"AUTH_DEV_ROLES":       "admin, user,,",
				"KEYCLOAK_ISSUER_URL":  "https://id.example.com/realms/app",
				"KEYCLOAK_AUDIENCE":    "api",
				"DATABASE_URL":         "postgres://app:secret@db:5432/app?sslmode=disable&pool_max_conns=8",
				"LISTEN_ADDR":          "127.0.0.1:8080",
				"MANAGEMENT_ADDR":      "[::1]:9090",
				"CORS_ALLOWED_ORIGINS": "https://app.example.com, http://localhost:3000",
				"LOG_LEVEL":            "debug",
			},
			want: Config{
				Environment:        "development",
				AuthProvider:       "keycloak",
				AuthDevRoles:       []string{"admin", "user"},
				KeycloakIssuerURL:  "https://id.example.com/realms/app",
				KeycloakAudience:   "api",
				DatabaseURL:        "postgres://app:secret@db:5432/app?sslmode=disable&pool_max_conns=8",
				ListenAddr:         "127.0.0.1:8080",
				ManagementAddr:     "[::1]:9090",
				CORSAllowedOrigins: []string{"https://app.example.com", "http://localhost:3000"},
				LogLevel:           slog.LevelDebug,
			},
		},
		{
			name: "the environment is trimmed and lower-cased",
			env:  map[string]string{"APP_ENVIRONMENT": " Dev "},
			want: Config{Environment: "dev", ListenAddr: ":8080"},
		},
		{
			name: "the provider none, the postgresql scheme and an http issuer are accepted",
			env: map[string]string{
				"AUTH_PROVIDER":       "none",
				"DATABASE_URL":        "postgresql://db/app",
				"KEYCLOAK_ISSUER_URL": "http://keycloak:8181/realms/app",
			},
			want: Config{
				Environment: "production", AuthProvider: "none", ListenAddr: ":8080",
				DatabaseURL: "postgresql://db/app", KeycloakIssuerURL: "http://keycloak:8181/realms/app",
			},
		},
		{
			name: "the log level is case-insensitive",
			env:  map[string]string{"LOG_LEVEL": "Warn"},
			want: Config{Environment: "production", ListenAddr: ":8080", LogLevel: slog.LevelWarn},
		},
		{
			name: "the highest port and the ERROR level",
			env:  map[string]string{"LISTEN_ADDR": "localhost:65535", "LOG_LEVEL": "ERROR"},
			want: Config{Environment: "production", ListenAddr: "localhost:65535", LogLevel: slog.LevelError},
		},
	}
	for _, tt := range valid {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parse(lookupIn(tt.env))

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	const (
		badIssuer = "is not valid: use an absolute http or https URL without query, fragment or trailing slash"
		badAddr   = "is not valid: use host:port with a port from 1 to 65535"
		badOrigin = "is not valid: use scheme://host[:port] with http or https"
	)
	invalid := []struct {
		name     string
		variable string
		value    string
		want     string
	}{
		{"an unknown provider", "AUTH_PROVIDER", "oidc", `AUTH_PROVIDER "oidc" is not supported: use "keycloak" or "none"`},
		{"the provider is matched exactly", "AUTH_PROVIDER", "Keycloak", `AUTH_PROVIDER "Keycloak" is not supported: use "keycloak" or "none"`},
		{"a relative issuer", "KEYCLOAK_ISSUER_URL", "realms/app", `KEYCLOAK_ISSUER_URL "realms/app" ` + badIssuer},
		{"an issuer with another scheme", "KEYCLOAK_ISSUER_URL", "ftp://id.example.com/realms/app", badIssuer},
		{"an issuer without a host", "KEYCLOAK_ISSUER_URL", "https:///realms/app", badIssuer},
		{"an issuer with a query", "KEYCLOAK_ISSUER_URL", "https://id.example.com/realms/app?x=1", badIssuer},
		{"an issuer with a fragment", "KEYCLOAK_ISSUER_URL", "https://id.example.com/realms/app#x", badIssuer},
		{"an issuer with a trailing slash", "KEYCLOAK_ISSUER_URL", "https://id.example.com/realms/app/", badIssuer},
		{"a database URL with another scheme", "DATABASE_URL", "mysql://app:secret@db/app", "DATABASE_URL is not valid: use a postgres:// or postgresql:// URL"},
		{"a database keyword string", "DATABASE_URL", "host=db user=app password=secret", "DATABASE_URL is not valid"},
		{"a listen address without a colon", "LISTEN_ADDR", "8080", `LISTEN_ADDR "8080" ` + badAddr},
		{"a listen address without a port", "LISTEN_ADDR", "localhost:", badAddr},
		{"a listen address with port zero", "LISTEN_ADDR", ":0", badAddr},
		{"a listen address with a port too high", "LISTEN_ADDR", ":65536", badAddr},
		{"a listen address with a named port", "LISTEN_ADDR", ":http", badAddr},
		{"a management address without a colon", "MANAGEMENT_ADDR", "9090", `MANAGEMENT_ADDR "9090" ` + badAddr},
		{"a management address with a port too high", "MANAGEMENT_ADDR", "127.0.0.1:70000", badAddr},
		{"the wildcard origin", "CORS_ALLOWED_ORIGINS", "*", `CORS_ALLOWED_ORIGINS "*" ` + badOrigin},
		{"a wildcard host", "CORS_ALLOWED_ORIGINS", "https://*.example.com", badOrigin},
		{"an origin with a trailing slash", "CORS_ALLOWED_ORIGINS", "https://app.example.com/", badOrigin},
		{"an origin with a path", "CORS_ALLOWED_ORIGINS", "https://app.example.com/app", badOrigin},
		{"an origin without a scheme", "CORS_ALLOWED_ORIGINS", "app.example.com", badOrigin},
		{"an origin with another scheme", "CORS_ALLOWED_ORIGINS", "ftp://app.example.com", badOrigin},
		{"one bad origin in a list", "CORS_ALLOWED_ORIGINS", "https://app.example.com,nope", `CORS_ALLOWED_ORIGINS "nope" ` + badOrigin},
		{"an unknown log level", "LOG_LEVEL", "TRACE", `LOG_LEVEL "TRACE" is not supported: use DEBUG, INFO, WARN or ERROR`},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parse(lookupIn(map[string]string{tt.variable: tt.value}))

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
			assert.True(t, strings.HasPrefix(err.Error(), tt.variable), "the message starts with the variable: %s", err)
			assert.NotContains(t, err.Error(), "secret")
			assert.Equal(t, Config{}, got, "a malformed value never yields a configuration")
		})
	}

	t.Run("every violation is reported in one error, one per line", func(t *testing.T) {
		_, err := parse(lookupIn(map[string]string{
			"AUTH_PROVIDER":        "oidc",
			"KEYCLOAK_ISSUER_URL":  "realms/app",
			"DATABASE_URL":         "mysql://db/app",
			"LISTEN_ADDR":          "8080",
			"MANAGEMENT_ADDR":      "9090",
			"CORS_ALLOWED_ORIGINS": "*, nope",
			"LOG_LEVEL":            "TRACE",
		}))

		require.Error(t, err)
		lines := strings.Split(err.Error(), "\n")
		require.Len(t, lines, 8)
		for i, prefix := range []string{
			"AUTH_PROVIDER ", "KEYCLOAK_ISSUER_URL ", "DATABASE_URL ", "LISTEN_ADDR ", "MANAGEMENT_ADDR ",
			`CORS_ALLOWED_ORIGINS "*"`, `CORS_ALLOWED_ORIGINS "nope"`, "LOG_LEVEL ",
		} {
			assert.True(t, strings.HasPrefix(lines[i], prefix), "line %d: %s", i, lines[i])
		}
	})
}

func TestValidate(t *testing.T) {
	const (
		database = "postgres://app:app@localhost:5434/app"
		refused  = "AUTH_PROVIDER=none is refused when APP_ENVIRONMENT is "
		allowed  = ": allowed only in development, dev, local, test"
	)
	none := func(environment string) map[string]string {
		env := map[string]string{"AUTH_PROVIDER": "none", "DATABASE_URL": database}
		if environment != "" {
			env["APP_ENVIRONMENT"] = environment
		}
		return env
	}
	tests := []struct {
		name string
		env  map[string]string
		want []string // the lines of the error; empty when the configuration is valid
	}{
		{
			name: "an unset provider is refused",
			env:  map[string]string{"DATABASE_URL": database},
			want: []string{`AUTH_PROVIDER is required: set "keycloak", or "none" for local development`},
		},
		{"none with an unset environment is refused", none(""), []string{refused + `"production"` + allowed}},
		{"none in production is refused", none("production"), []string{refused + `"production"` + allowed}},
		{"none in prod is refused", none("prod"), []string{refused + `"prod"` + allowed}},
		{"none in Production is refused", none("Production"), []string{refused + `"production"` + allowed}},
		{"none in staging is refused", none("staging"), []string{refused + `"staging"` + allowed}},
		{"none in development is accepted", none("development"), nil},
		{"none in dev is accepted", none("dev"), nil},
		{"none in local is accepted", none("local"), nil},
		{"none in test is accepted", none("test"), nil},
		{"none in a padded, mixed-case dev is accepted", none(" Dev "), nil},
		{
			name: "keycloak with issuer and audience is accepted in any environment",
			env: map[string]string{
				"AUTH_PROVIDER": "keycloak", "DATABASE_URL": database,
				"KEYCLOAK_ISSUER_URL": "https://id.example.com/realms/app", "KEYCLOAK_AUDIENCE": "api",
			},
			want: nil,
		},
		{
			name: "keycloak without an issuer is refused",
			env:  map[string]string{"AUTH_PROVIDER": "keycloak", "DATABASE_URL": database, "KEYCLOAK_AUDIENCE": "api"},
			want: []string{"KEYCLOAK_ISSUER_URL is required when AUTH_PROVIDER=keycloak"},
		},
		{
			name: "keycloak without an audience is refused",
			env: map[string]string{
				"AUTH_PROVIDER": "keycloak", "DATABASE_URL": database,
				"KEYCLOAK_ISSUER_URL": "https://id.example.com/realms/app",
			},
			want: []string{"KEYCLOAK_AUDIENCE is required when AUTH_PROVIDER=keycloak"},
		},
		{
			name: "an unset database URL is refused",
			env:  map[string]string{"AUTH_PROVIDER": "none", "APP_ENVIRONMENT": "dev"},
			want: []string{"DATABASE_URL is required"},
		},
		{
			name: "equal addresses are refused",
			env: map[string]string{
				"AUTH_PROVIDER": "none", "APP_ENVIRONMENT": "dev", "DATABASE_URL": database,
				"LISTEN_ADDR": "127.0.0.1:8080", "MANAGEMENT_ADDR": "127.0.0.1:8080",
			},
			want: []string{"MANAGEMENT_ADDR must differ from LISTEN_ADDR"},
		},
		{
			name: "different addresses are accepted",
			env: map[string]string{
				"AUTH_PROVIDER": "none", "APP_ENVIRONMENT": "dev", "DATABASE_URL": database,
				"LISTEN_ADDR": "127.0.0.1:8080", "MANAGEMENT_ADDR": "127.0.0.1:9090",
			},
			want: nil,
		},
		{
			name: "every violation is reported in one error",
			env:  map[string]string{"AUTH_PROVIDER": "keycloak", "LISTEN_ADDR": ":8080", "MANAGEMENT_ADDR": ":8080"},
			want: []string{
				"KEYCLOAK_ISSUER_URL is required when AUTH_PROVIDER=keycloak",
				"KEYCLOAK_AUDIENCE is required when AUTH_PROVIDER=keycloak",
				"DATABASE_URL is required",
				"MANAGEMENT_ADDR must differ from LISTEN_ADDR",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parse(lookupIn(tt.env))
			require.NoError(t, err)

			err = cfg.Validate()

			if len(tt.want) == 0 {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.want, strings.Split(err.Error(), "\n"))
		})
	}
}

func TestLoad_EnvironmentWinsOverDotenv(t *testing.T) {
	isolate(t)
	writeDotenv(t, "AUTH_PROVIDER=none\nLISTEN_ADDR=127.0.0.1:1111\nLOG_LEVEL=DEBUG\n")
	t.Setenv("LISTEN_ADDR", "127.0.0.1:2222")

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:2222", cfg.ListenAddr, "the process environment wins")
	assert.Equal(t, "none", cfg.AuthProvider, "the file fills what the environment leaves unset")
	assert.Equal(t, slog.LevelDebug, cfg.LogLevel, "an empty environment value counts as unset")
	assert.Equal(t, "production", cfg.Environment, "the default fills what neither sets")
	assert.Empty(t, os.Getenv("AUTH_PROVIDER"), "Load does not change the process environment")
}

func TestLoad_NoFile(t *testing.T) {
	isolate(t)

	cfg, err := Load()

	require.NoError(t, err)
	assert.Equal(t, Config{Environment: "production", ListenAddr: ":8080", LogLevel: slog.LevelInfo}, cfg)
}

func TestDotenv(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]string
		wantErr string
	}{
		{
			name:    "pairs, comments and blank lines",
			content: "# the dev environment\n\nA=1\n  # indented comment\nB=two words\n",
			want:    map[string]string{"A": "1", "B": "two words"},
		},
		{
			name:    "one pair of matching quotes is removed",
			content: "A=\"double\"\nB='single'\nC=\"'both'\"\nD=\"open\nE=\"mixed'\nF=\"\"\n",
			want:    map[string]string{"A": "double", "B": "single", "C": "'both'", "D": "\"open", "E": "\"mixed'", "F": ""},
		},
		{
			name:    "the value keeps later equals signs and surrounding space is trimmed",
			content: "URL = postgres://app:app@localhost:5434/app?sslmode=disable \r\nEMPTY=\n",
			want:    map[string]string{"URL": "postgres://app:app@localhost:5434/app?sslmode=disable", "EMPTY": ""},
		},
		{
			name:    "no interpolation and no inline comment",
			content: "A=$HOME\nB=${A}\nC=x # y\n",
			want:    map[string]string{"A": "$HOME", "B": "${A}", "C": "x # y"},
		},
		{
			name:    "the last of two equal keys wins",
			content: "A=1\nA=2\n",
			want:    map[string]string{"A": "2"},
		},
		{
			name:    "a line without an equals sign is an error",
			content: "A=1\n\nnonsense\n",
			wantErr: ".env.local line 3: want KEY=value",
		},
		{
			name:    "an export prefix is an error",
			content: "export A=1\n",
			wantErr: ".env.local line 1: want KEY=value",
		},
		{
			name:    "an empty key is an error",
			content: "# c\n=1\n",
			wantErr: ".env.local line 2: want KEY=value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolate(t)
			writeDotenv(t, tt.content)

			got, err := dotenv()

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("a missing file yields an empty map", func(t *testing.T) {
		isolate(t)

		got, err := dotenv()

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("a malformed file fails Load", func(t *testing.T) {
		isolate(t)
		writeDotenv(t, "nonsense\n")

		_, err := Load()

		require.EqualError(t, err, ".env.local line 1: want KEY=value")
	})
}

var (
	// a variable in .env.example is a line that sets it or a commented line that would
	envExampleRe = regexp.MustCompile(`(?m)^#?([A-Z][A-Z0-9_]*)=`)
	// a variable in the reference is the first cell of a table row
	envReferenceRe = regexp.MustCompile("(?m)^\\|\\s*`([A-Z][A-Z0-9_]*)`\\s*\\|")
)

func TestEnvFilesListEveryVariable(t *testing.T) {
	tests := []struct {
		path string
		re   *regexp.Regexp
	}{
		{"../../../.env.example", envExampleRe},
		{"../../../docs/agents/env-reference.md", envReferenceRe},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			content, err := os.ReadFile(tt.path)
			require.NoError(t, err)

			var names []string
			for _, m := range tt.re.FindAllSubmatch(content, -1) {
				names = append(names, string(m[1]))
			}

			assert.ElementsMatch(t, variables, names, "each variable the reader knows appears exactly once, and no other")
		})
	}
}
