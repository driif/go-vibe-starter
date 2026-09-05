package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/driif/go-vibe-starter/internal/server/config"
)

func TestAuthDefaultsFromEnv(t *testing.T) {
	cfg := config.DefaultServiceConfigFromEnv()

	assert.Equal(t, config.AuthProviderNone, cfg.Auth.Provider)
	assert.Equal(t, "local-dev", cfg.Auth.DevSubject)
	assert.Equal(t, []string{"admin"}, cfg.Auth.DevRoles)
	require.NoError(t, cfg.Validate())
}

func TestAuthFromEnv(t *testing.T) {
	t.Setenv("AUTH_PROVIDER", config.AuthProviderKeycloak)
	t.Setenv("AUTH_DEV_SUBJECT", "svc-account")
	t.Setenv("AUTH_DEV_ROLES", "admin, editor ,viewer")

	cfg := config.DefaultServiceConfigFromEnv()

	assert.Equal(t, config.AuthProviderKeycloak, cfg.Auth.Provider)
	assert.Equal(t, "svc-account", cfg.Auth.DevSubject)
	assert.Equal(t, []string{"admin", "editor", "viewer"}, cfg.Auth.DevRoles)
}

func TestAppValidate(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		provider    string
		wantErr     string
	}{
		{
			name:        "none in development",
			environment: "development",
			provider:    config.AuthProviderNone,
		},
		{
			name:        "keycloak in production",
			environment: config.EnvironmentProduction,
			provider:    config.AuthProviderKeycloak,
		},
		{
			name:        "none in production",
			environment: config.EnvironmentProduction,
			provider:    config.AuthProviderNone,
			wantErr:     `AUTH_PROVIDER=none is not allowed with APP_ENVIRONMENT=production: set AUTH_PROVIDER=keycloak`,
		},
		{
			name:        "unknown provider",
			environment: "development",
			provider:    "auth0",
			wantErr:     `invalid AUTH_PROVIDER "auth0": accepted values are "none" and "keycloak"`,
		},
		{
			name:        "empty provider",
			environment: "development",
			provider:    "",
			wantErr:     `invalid AUTH_PROVIDER "": accepted values are "none" and "keycloak"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.App{
				Environment: tt.environment,
				Auth:        config.Auth{Provider: tt.provider},
			}

			err := cfg.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Equal(t, tt.wantErr, err.Error())
		})
	}
}
