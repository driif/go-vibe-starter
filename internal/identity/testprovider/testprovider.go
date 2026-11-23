// Package testprovider supplies independent discovery, signing keys and tokens for tests.
package testprovider

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Audience is the audience added to fixture access tokens.
	Audience = "api"
	// KeyID identifies the fixture signing key.
	KeyID = "test"
)

var signingKey = sync.OnceValue(func() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return key
})

// Provider is an in-process issuer with a signing key shared across the test process.
type Provider struct {
	server   *httptest.Server
	down     atomic.Bool
	requests atomic.Int64
}

// New starts the fixture issuer.
func New() *Provider {
	p := &Provider{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": p.URL(), "jwks_uri": p.URL() + "/protocol/openid-connect/certs",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("GET /protocol/openid-connect/certs", func(w http.ResponseWriter, _ *http.Request) {
		key := signingKey().PublicKey
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{
			"kty": "RSA", "alg": "RS256", "use": "sig", "kid": KeyID,
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	})
	p.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.requests.Add(1)
		if p.down.Load() {
			http.Error(w, "provider unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		mux.ServeHTTP(w, r)
	}))
	return p
}

// URL returns the issuer URL.
func (p *Provider) URL() string { return p.server.URL }

// Close stops the fixture issuer.
func (p *Provider) Close() { p.server.Close() }

// SetDown makes all endpoints answer service unavailable when down is true.
func (p *Provider) SetDown(down bool) { p.down.Store(down) }

// Requests reports how many HTTP requests the provider received.
func (p *Provider) Requests() int64 { return p.requests.Load() }

// Claims returns a valid Keycloak-shaped access-token payload.
func (p *Provider) Claims(subject string, roles ...string) map[string]any {
	now := time.Now()
	return map[string]any{
		"iss": p.URL(), "sub": subject, "aud": Audience, "azp": "app-test", "typ": "Bearer",
		"iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix(),
		"preferred_username": subject, "email": subject + "@example.test", "name": "Test " + subject,
		"realm_access": map[string]any{"roles": roles},
	}
}

// Token signs claims with the fixture's standard JWT header.
func (p *Provider) Token(claims map[string]any) string {
	return p.Sign(map[string]any{"alg": "RS256", "typ": "JWT", "kid": KeyID}, claims)
}

// Sign signs any header and claims with RSA-SHA256, including intentionally invalid headers.
func (*Provider) Sign(header, claims map[string]any) string {
	encode := func(value map[string]any) string {
		data, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		return base64.RawURLEncoding.EncodeToString(data)
	}
	input := encode(header) + "." + encode(claims)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, signingKey(), crypto.SHA256, digest[:])
	if err != nil {
		panic(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}
