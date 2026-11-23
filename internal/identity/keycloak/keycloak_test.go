package keycloak

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/driif/go-vibe-starter/internal/identity"
	"github.com/driif/go-vibe-starter/internal/identity/testprovider"
)

func TestVerify(t *testing.T) {
	p := testprovider.New()
	t.Cleanup(p.Close)
	v := New(p.URL(), testprovider.Audience)
	tests := []struct {
		name   string
		change func(map[string]any, map[string]any)
		token  func(string) string
		accept bool
	}{
		{name: "Keycloak access token", accept: true},
		{name: "audience list", change: func(_ map[string]any, c map[string]any) { c["aud"] = []string{"other", "api"} }, accept: true},
		{name: "at+jwt header", change: func(h, c map[string]any) { h["typ"] = "at+jwt"; delete(c, "typ") }, accept: true},
		{name: "application/at+jwt header", change: func(h, c map[string]any) { h["typ"] = "application/at+jwt"; delete(c, "typ") }, accept: true},
		{name: "no realm roles", change: func(_ map[string]any, c map[string]any) { delete(c, "realm_access") }, accept: true},
		{name: "expired", change: func(_ map[string]any, c map[string]any) { c["exp"] = time.Now().Add(-time.Minute).Unix() }},
		{name: "no expiry", change: func(_ map[string]any, c map[string]any) { delete(c, "exp") }},
		{name: "not yet valid", change: func(_ map[string]any, c map[string]any) { c["nbf"] = time.Now().Add(10 * time.Minute).Unix() }},
		{name: "wrong audience", change: func(_ map[string]any, c map[string]any) { c["aud"] = "other" }},
		{name: "no audience", change: func(_ map[string]any, c map[string]any) { delete(c, "aud") }},
		{name: "wrong issuer", change: func(_ map[string]any, c map[string]any) { c["iss"] = "https://other.example" }},
		{name: "no subject", change: func(_ map[string]any, c map[string]any) { delete(c, "sub") }},
		{name: "empty subject", change: func(_ map[string]any, c map[string]any) { c["sub"] = "" }},
		{name: "ID token", change: func(_ map[string]any, c map[string]any) { c["typ"] = "ID" }},
		{name: "ID token with access-token header", change: func(h, c map[string]any) { c["typ"] = "ID"; h["typ"] = "at+jwt" }},
		{name: "refresh token", change: func(_ map[string]any, c map[string]any) { c["typ"] = "Refresh" }},
		{name: "DPoP token", change: func(_ map[string]any, c map[string]any) { c["typ"] = "DPoP" }},
		{name: "null token type with access-token header", change: func(h, c map[string]any) { c["typ"] = nil; h["typ"] = "at+jwt" }},
		{name: "no access-token type", change: func(h, c map[string]any) { delete(h, "typ"); delete(c, "typ") }},
		{name: "unknown key", change: func(h, _ map[string]any) { h["kid"] = "unknown" }},
		{name: "symmetric algorithm", change: func(h, _ map[string]any) { h["alg"] = "HS256" }},
		{name: "unsigned algorithm", change: func(h, _ map[string]any) { h["alg"] = "none" }},
		{name: "tampered signature", token: func(raw string) string {
			parts := strings.Split(raw, ".")
			signature, err := base64.RawURLEncoding.DecodeString(parts[2])
			require.NoError(t, err)
			signature[0] ^= 0xff
			parts[2] = base64.RawURLEncoding.EncodeToString(signature)
			return strings.Join(parts, ".")
		}},
		{name: "malformed token", token: func(string) string { return "not-a-jwt" }},
		{name: "empty token", token: func(string) string { return "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := p.Claims("alice", "user")
			header := map[string]any{"alg": "RS256", "typ": "JWT", "kid": testprovider.KeyID}
			if tt.change != nil {
				tt.change(header, claims)
			}
			raw := p.Sign(header, claims)
			if tt.token != nil {
				raw = tt.token(raw)
			}
			principal, err := v.Verify(t.Context(), raw)
			if tt.accept {
				require.NoError(t, err)
				require.NotNil(t, principal)
				assert.Equal(t, "alice", principal.Subject)
			} else {
				require.ErrorIs(t, err, identity.ErrUnauthenticated)
				assert.NotErrorIs(t, err, identity.ErrUnavailable)
				assert.Nil(t, principal)
			}
		})
	}
}

func TestVerify_Principal(t *testing.T) {
	p := testprovider.New()
	t.Cleanup(p.Close)
	v := New(p.URL(), testprovider.Audience)
	claims := p.Claims("alice", "user", "admin", "default-roles-app")
	principal, err := v.Verify(t.Context(), p.Token(claims))
	require.NoError(t, err)
	assert.Equal(t, &identity.Principal{
		Subject: "alice", Username: "alice", Email: "alice@example.test", Name: "Test alice",
		Roles: []string{"user", "admin", "default-roles-app"},
	}, principal)
}

func TestVerify_ProviderDownAtStart(t *testing.T) {
	p := testprovider.New()
	t.Cleanup(p.Close)
	p.SetDown(true)
	v := New(p.URL(), testprovider.Audience)
	assert.Zero(t, p.Requests(), "construction must not perform I/O")
	_, err := v.Verify(t.Context(), p.Token(p.Claims("alice")))
	require.ErrorIs(t, err, identity.ErrUnavailable)
	assert.NotErrorIs(t, err, identity.ErrUnauthenticated)
}

func TestVerify_ProviderUnreachable(t *testing.T) {
	p := testprovider.New()
	raw := p.Token(p.Claims("alice"))
	v := New(p.URL(), testprovider.Audience)
	p.Close()
	_, err := v.Verify(t.Context(), raw)
	require.ErrorIs(t, err, identity.ErrUnavailable)
}

func TestVerify_ProviderDownLater(t *testing.T) {
	p := testprovider.New()
	t.Cleanup(p.Close)
	v := New(p.URL(), testprovider.Audience)
	r := &replay{base: http.DefaultTransport, ttl: 5 * time.Second}
	v.keys.ctx = oidc.ClientContext(context.Background(), &http.Client{Transport: r, Timeout: fetchTimeout})
	raw := p.Token(p.Claims("alice"))
	_, err := v.Verify(t.Context(), raw)
	require.NoError(t, err)
	p.SetDown(true)
	requests := p.Requests()
	_, err = v.Verify(t.Context(), raw)
	require.NoError(t, err)
	assert.Equal(t, requests, p.Requests())

	r.mu.Lock()
	clear(r.cache)
	r.mu.Unlock()
	_, err = v.Verify(t.Context(), p.Sign(map[string]any{"alg": "RS256", "kid": "unknown"}, p.Claims("alice")))
	require.ErrorIs(t, err, identity.ErrUnavailable)
	_, err = v.Verify(t.Context(), raw)
	require.NoError(t, err, "failed refresh must retain previously trusted signing keys")
}

func TestVerify_WrongIssuer(t *testing.T) {
	p := testprovider.New()
	t.Cleanup(p.Close)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issuer":"` + p.URL() + `","jwks_uri":"` + p.URL() + `/protocol/openid-connect/certs"}`))
		}
	}))
	t.Cleanup(proxy.Close)
	v := New(proxy.URL, testprovider.Audience)
	require.ErrorIs(t, v.Ready(t.Context()), ErrIssuerMismatch)
	_, err := v.Verify(t.Context(), p.Token(p.Claims("alice")))
	require.ErrorIs(t, err, identity.ErrUnavailable)
}

func TestVerify_ForgedTokenCost(t *testing.T) {
	p := testprovider.New()
	t.Cleanup(p.Close)
	v := New(p.URL(), testprovider.Audience)
	header := map[string]any{"alg": "RS256", "kid": "unknown"}
	raw := p.Sign(header, p.Claims("alice"))
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			_, err := v.Verify(context.Background(), raw)
			assert.ErrorIs(t, err, identity.ErrUnauthenticated)
		})
	}
	wg.Wait()
	assert.EqualValues(t, 2, p.Requests(), "one discovery and one JWKS fetch")
}

func TestReady(t *testing.T) {
	p := testprovider.New()
	t.Cleanup(p.Close)
	p.SetDown(true)
	v := New(p.URL(), testprovider.Audience)
	r := &replay{base: http.DefaultTransport, ttl: 5 * time.Second}
	v.keys.ctx = oidc.ClientContext(context.Background(), &http.Client{Transport: r, Timeout: fetchTimeout})
	require.Error(t, v.Ready(t.Context()))
	p.SetDown(false)
	r.mu.Lock()
	clear(r.cache)
	r.mu.Unlock()
	require.NoError(t, v.Ready(t.Context()))
	p.SetDown(true)
	requests := p.Requests()
	_, err := v.Verify(t.Context(), p.Token(p.Claims("alice")))
	require.NoError(t, err)
	require.NoError(t, v.Ready(t.Context()))
	assert.Equal(t, requests, p.Requests())
}
