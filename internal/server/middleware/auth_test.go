package middleware_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/driif/go-vibe-starter/internal/identity"
	"github.com/driif/go-vibe-starter/internal/reqlog"
	"github.com/driif/go-vibe-starter/internal/server/middleware"
)

type verifierFunc func(context.Context, string) (*identity.Principal, error)

func (fn verifierFunc) Verify(ctx context.Context, token string) (*identity.Principal, error) {
	return fn(ctx, token)
}

func TestAuth(t *testing.T) {
	for _, tt := range []struct {
		name, header, token, challenge, code string
		err                                  error
		principal                            *identity.Principal
		status                               int
	}{
		{name: "valid", header: "Bearer token", token: "token", principal: &identity.Principal{Subject: "alice"}, status: 200},
		{name: "case insensitive", header: "bEaReR token", token: "token", principal: &identity.Principal{Subject: "alice"}, status: 200},
		{name: "missing", challenge: "Bearer", code: "unauthenticated", err: identity.ErrUnauthenticated, status: 401},
		{name: "other scheme", header: "Basic token", challenge: "Bearer", code: "unauthenticated", err: identity.ErrUnauthenticated, status: 401},
		{name: "empty", header: "Bearer ", challenge: "Bearer", code: "unauthenticated", err: identity.ErrUnauthenticated, status: 401},
		{name: "rejected", header: "Bearer token", token: "token", challenge: `Bearer error="invalid_token"`, code: "unauthenticated", err: identity.ErrUnauthenticated, status: 401},
		{name: "outage", header: "Bearer token", token: "token", code: "unavailable", err: identity.ErrUnavailable, status: 503},
		{name: "unknown error", header: "Bearer token", token: "token", challenge: `Bearer error="invalid_token"`, code: "unauthenticated", err: errors.New("private"), status: 401},
		{name: "nil principal", challenge: "Bearer", code: "unauthenticated", status: 401},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := reqlog.With(t.Context(), "id")
			r := httptest.NewRequest(http.MethodPost, "/private", strings.NewReader("body")).WithContext(ctx)
			r.Header.Set("Authorization", tt.header)
			called := false
			v := verifierFunc(func(_ context.Context, token string) (*identity.Principal, error) {
				require.Equal(t, tt.token, token)
				return tt.principal, tt.err
			})
			h := middleware.Auth(v)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				require.Same(t, tt.principal, identity.FromContext(r.Context()))
				w.WriteHeader(http.StatusOK)
			}))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			require.Equal(t, tt.status, w.Code)
			require.Equal(t, tt.challenge, w.Header().Get("WWW-Authenticate"))
			require.Equal(t, tt.code, reqlog.Code(ctx))
			require.Equal(t, tt.status == 200, called)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Equal(t, "body", string(body))
			require.NotContains(t, w.Body.String(), "private")
		})
	}
}
