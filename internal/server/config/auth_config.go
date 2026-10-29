package config

import (
	"fmt"
	"sort"
	"strings"
)

// Accepted values for AUTH_PROVIDER.
const (
	// AuthProviderNone skips token verification and runs every request as the dev principal.
	AuthProviderNone = "none"
	// AuthProviderKeycloak verifies bearer tokens against the configured Keycloak realm.
	AuthProviderKeycloak = "keycloak"
)

// EnvironmentProduction is the conventional APP_ENVIRONMENT value for a deployed service.
const EnvironmentProduction = "production"

// unverifiedAuthEnvironments are the APP_ENVIRONMENT values allowed to run without token
// verification. The check is an allow-list rather than a production deny-list so that a typo
// ("prod", "Production", "staging") fails closed and refuses to start, instead of silently
// serving every request as the dev principal.
var unverifiedAuthEnvironments = map[string]bool{
	"development": true,
	"dev":         true,
	"local":       true,
	"test":        true,
}

func allowedUnverifiedEnvironments() string {
	names := make([]string, 0, len(unverifiedAuthEnvironments))
	for name := range unverifiedAuthEnvironments {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// Auth selects how incoming requests are authenticated.
type Auth struct {
	Provider string
	// DevSubject and DevRoles describe the principal injected when Provider is AuthProviderNone.
	DevSubject string
	DevRoles   []string
}

// Validate reports configuration that would start the server in an unsafe or undefined state.
// The bootstrap calls it before building the server and aborts on error.
func (a App) Validate() error {
	switch a.Auth.Provider {
	case AuthProviderKeycloak:
	case AuthProviderNone:
		if !unverifiedAuthEnvironments[strings.ToLower(strings.TrimSpace(a.Environment))] {
			return fmt.Errorf(
				"AUTH_PROVIDER=%s is only allowed when APP_ENVIRONMENT is one of [%s], got %q: set AUTH_PROVIDER=%s",
				AuthProviderNone, allowedUnverifiedEnvironments(), a.Environment, AuthProviderKeycloak,
			)
		}
	default:
		return fmt.Errorf(
			"invalid AUTH_PROVIDER %q: accepted values are %q and %q",
			a.Auth.Provider, AuthProviderNone, AuthProviderKeycloak,
		)
	}

	return nil
}
