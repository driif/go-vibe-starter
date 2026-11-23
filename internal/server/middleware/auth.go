package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	"github.com/driif/go-vibe-starter/internal/identity"
	"github.com/driif/go-vibe-starter/internal/reqlog"
)

// Verifier checks a bearer token and returns its authenticated principal.
type Verifier interface {
	Verify(context.Context, string) (*identity.Principal, error)
}

// Auth authenticates before reading any request body and writes protocol-aware failures.
func Auth(v Verifier) func(http.Handler) http.Handler {
	writer := connecthttp.NewErrorWriter()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string
			if scheme, value, ok := strings.Cut(r.Header.Get("Authorization"), " "); ok && strings.EqualFold(scheme, "Bearer") {
				token = strings.TrimSpace(value)
			}
			p, err := v.Verify(r.Context(), token)
			if err == nil && p != nil {
				next.ServeHTTP(w, r.WithContext(identity.WithPrincipal(r.Context(), p)))
				return
			}
			code, message := connect.CodeUnauthenticated, "invalid or missing bearer token"
			if errors.Is(err, identity.ErrUnavailable) {
				code, message = connect.CodeUnavailable, "identity provider unavailable"
				slog.ErrorContext(r.Context(), "identity provider unavailable", "error", err)
			} else {
				challenge := "Bearer"
				if token != "" {
					challenge += ` error="invalid_token"`
				}
				w.Header().Set("WWW-Authenticate", challenge)
				slog.DebugContext(r.Context(), "token rejected", "error", err)
			}
			reqlog.SetCode(r.Context(), code.String())
			if err := writer.Write(w, r, connect.NewError(code, message)); err != nil {
				slog.DebugContext(r.Context(), "write authentication response", "error", err)
			}
		})
	}
}
