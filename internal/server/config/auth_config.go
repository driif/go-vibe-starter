package config

import "fmt"

// Accepted values for AUTH_PROVIDER.
const (
	// AuthProviderNone skips token verification and runs every request as the dev principal.
	AuthProviderNone = "none"
	// AuthProviderKeycloak verifies bearer tokens against the configured Keycloak realm.
	AuthProviderKeycloak = "keycloak"
)

// EnvironmentProduction is the APP_ENVIRONMENT value that turns on the strict startup checks.
const EnvironmentProduction = "production"

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
		if a.Environment == EnvironmentProduction {
			return fmt.Errorf(
				"AUTH_PROVIDER=%s is not allowed with APP_ENVIRONMENT=%s: set AUTH_PROVIDER=%s",
				AuthProviderNone, EnvironmentProduction, AuthProviderKeycloak,
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
