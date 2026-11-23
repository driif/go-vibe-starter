// Package identity is the caller of a request. Code that reads a Principal does not know which
// identity provider issued the token.
package identity

import (
	"context"
	"errors"
	"slices"
)

var (
	// ErrUnauthenticated means the token was not accepted.
	ErrUnauthenticated = errors.New("identity: unauthenticated")
	// ErrUnavailable means the token could not be checked because the identity provider did not
	// answer.
	ErrUnavailable = errors.New("identity: provider unavailable")
)

// Principal is the authenticated caller.
type Principal struct {
	Subject  string   // "sub"; the owner id rows are scoped by
	Username string   // "preferred_username"
	Email    string   // "email"
	Name     string   // "name"
	Roles    []string // "realm_access.roles", unfiltered
}

// HasRole reports whether the principal has the realm role. It is false for a nil principal.
func (p *Principal) HasRole(role string) bool {
	return p != nil && slices.Contains(p.Roles, role)
}

type principalKey struct{}

// WithPrincipal returns ctx carrying p.
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// FromContext returns the principal the auth middleware stored, or nil when there is none.
func FromContext(ctx context.Context) *Principal {
	p, _ := ctx.Value(principalKey{}).(*Principal)
	return p
}

// DevVerifier accepts every request as the dev principal. It is the verifier of
// AUTH_PROVIDER=none.
type DevVerifier struct{ Roles []string }

// Verify ignores the token and returns a new principal with subject and username local-dev and
// a copy of Roles. It never returns an error.
func (d DevVerifier) Verify(context.Context, string) (*Principal, error) {
	return &Principal{
		Subject:  "local-dev",
		Username: "local-dev",
		Roles:    slices.Clone(d.Roles),
	}, nil
}
