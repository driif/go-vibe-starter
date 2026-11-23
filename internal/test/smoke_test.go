//go:build smoke

package test_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv1 "github.com/driif/go-vibe-starter/internal/gen/api/v1"
	"github.com/driif/go-vibe-starter/internal/gen/api/v1/apiv1connect"
	"github.com/driif/go-vibe-starter/internal/identity/keycloak"
	"github.com/driif/go-vibe-starter/internal/server"
	"github.com/driif/go-vibe-starter/internal/test"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
}

func TestSmoke_WhoAmI(t *testing.T) {
	issuer := os.Getenv("KEYCLOAK_ISSUER_URL")
	if issuer == "" {
		issuer = "http://localhost:8181/realms/app"
	}
	verifier := keycloak.New(issuer, "api")
	require.NoError(t, verifier.Ready(t.Context()), "Keycloak is not reachable: run make keycloak-up")
	s := test.NewServer(t, func(d *server.Deps) { d.Verifier = verifier })
	alice := realmToken(t, issuer, url.Values{"grant_type": {"password"}, "client_id": {"app-test"}, "username": {"alice"}, "password": {"password"}, "scope": {"openid"}})
	bob := realmToken(t, issuer, url.Values{"grant_type": {"password"}, "client_id": {"app-test"}, "username": {"bob"}, "password": {"password"}, "scope": {"openid"}})
	service := realmToken(t, issuer, url.Values{"grant_type": {"client_credentials"}, "client_id": {"app-service"}, "client_secret": {"dev-only-secret"}})
	require.NotEmpty(t, alice.IDToken)
	require.NotEmpty(t, alice.RefreshToken)
	for _, tt := range []struct {
		name, token, username, displayName string
		accepted                           bool
		admin                              bool
	}{
		{"alice", alice.AccessToken, "alice", "Alice Admin", true, true},
		{"bob", bob.AccessToken, "bob", "Bob User", true, false},
		{"service", service.AccessToken, "service-account-app-service", "", true, false},
		{"ID token", alice.IDToken, "", "", false, false},
		{"refresh token", alice.RefreshToken, "", "", false, false},
		{"no token", "", "", "", false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := apiv1connect.NewWhoAmIServiceClient(s.Client(tt.token))
			res, err := client.WhoAmI(t.Context(), &apiv1.WhoAmIRequest{})
			if !tt.accepted {
				require.Error(t, err)
				assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, res.GetSubject())
			assert.Equal(t, tt.username, res.GetUsername())
			assert.Equal(t, tt.displayName, res.GetName())
			if tt.name == "service" {
				assert.Empty(t, res.GetEmail())
				return
			}
			assert.Equal(t, tt.name+"@example.test", res.GetEmail())
			assert.Contains(t, res.GetRoles(), "user")
			if tt.admin {
				assert.Contains(t, res.GetRoles(), "admin")
			} else {
				assert.NotContains(t, res.GetRoles(), "admin")
			}
		})
	}
}

func realmToken(t *testing.T, issuer string, values url.Values) tokenResponse {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, issuer+"/protocol/openid-connect/token", strings.NewReader(values.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode, "dev realm token grant failed")
	var tokens tokenResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&tokens))
	require.NotEmpty(t, tokens.AccessToken)
	return tokens
}
