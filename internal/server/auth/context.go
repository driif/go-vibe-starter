package auth

import (
	"context"

	"github.com/driif/go-vibe-starter/pkg/keycloak"
)

type contextKey string

const (
	principalContextKey contextKey = "auth.principal"
	tokenContextKey     contextKey = "auth.token"
)

func withPrincipal(ctx context.Context, principal *keycloak.Principal, token string) context.Context {
	ctx = context.WithValue(ctx, principalContextKey, principal)
	ctx = context.WithValue(ctx, tokenContextKey, token)
	return ctx
}

func PrincipalFromContext(ctx context.Context) (*keycloak.Principal, bool) {
	value := ctx.Value(principalContextKey)
	principal, ok := value.(*keycloak.Principal)
	if !ok || principal == nil {
		return nil, false
	}
	return principal, true
}

func TokenFromContext(ctx context.Context) (string, bool) {
	value := ctx.Value(tokenContextKey)
	token, ok := value.(string)
	return token, ok && token != ""
}
