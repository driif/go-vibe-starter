package auth

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/driif/go-vibe-starter/pkg/keycloak"
)

type Mode string

const (
	ModeRequired Mode = "required"
	ModeOptional Mode = "optional"
)

type TokenVerifier interface {
	Verify(context.Context, string) (*keycloak.Principal, error)
}

type Options struct {
	Mode        Mode
	HeaderName  string
	TokenScheme string
}

func Authenticate(verifier TokenVerifier, opts Options) func(http.Handler) http.Handler {
	if verifier == nil {
		panic("auth: verifier is required")
	}

	if opts.Mode == "" {
		opts.Mode = ModeRequired
	}
	if opts.HeaderName == "" {
		opts.HeaderName = "Authorization"
	}
	if opts.TokenScheme == "" {
		opts.TokenScheme = "Bearer"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := bearerToken(r.Header.Get(opts.HeaderName), opts.TokenScheme)
			if err != nil {
				if errors.Is(err, ErrMissingBearerToken) && opts.Mode == ModeOptional {
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			principal, err := verifier.Verify(r.Context(), token)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), principal, token)))
		})
	}
}

func RequireRealmRoles(all bool, roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			if !matchRoles(all, roles, principal.HasRealmRole) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireClientRoles(clientID string, all bool, roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			if !matchRoles(all, roles, func(role string) bool {
				return principal.HasClientRole(clientID, role)
			}) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireOrganization() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			if len(principal.Organizations) == 0 {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAnyOrganization(orgs ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			if !slices.ContainsFunc(orgs, principal.HasOrganization) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(value string, scheme string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrMissingBearerToken
	}

	actualScheme, token, ok := strings.Cut(value, " ")
	if !ok || !strings.EqualFold(actualScheme, scheme) {
		return "", ErrMalformedToken
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", ErrMalformedToken
	}

	return token, nil
}

func matchRoles(all bool, required []string, hasRole func(string) bool) bool {
	if len(required) == 0 {
		return true
	}

	if all {
		for _, role := range required {
			if !hasRole(role) {
				return false
			}
		}
		return true
	}

	return slices.ContainsFunc(required, hasRole)
}
