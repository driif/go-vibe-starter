package auth

import (
	"net/http"

	"github.com/driif/go-vibe-starter/pkg/keycloak"
)

// DevPrincipal injects a fixed principal into every request, without looking at
// the Authorization header. It backs AUTH_PROVIDER=none so handlers keep the
// same contract as with a real token verifier.
func DevPrincipal(subject string, realmRoles []string) func(http.Handler) http.Handler {
	// built once and shared: the middleware never mutates it.
	principal := &keycloak.Principal{
		Subject:    subject,
		Username:   subject,
		RealmRoles: append([]string(nil), realmRoles...),
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), principal, "")))
		})
	}
}
