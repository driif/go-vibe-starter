// Package keycloak verifies access tokens with the issuer's discovered signing keys.
package keycloak

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"golang.org/x/sync/singleflight"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/driif/go-vibe-starter/internal/identity"
)

// ErrIssuerMismatch means discovery reports a different issuer than configured.
var ErrIssuerMismatch = errors.New("keycloak: provider reports another issuer")

const fetchTimeout = 5 * time.Second

// Verifier accepts signed access tokens for one issuer and audience.
type Verifier struct {
	verifier *oidc.IDTokenVerifier
	keys     *keySet
}

// New constructs a verifier without contacting the provider.
func New(issuer, audience string) *Verifier {
	transport := &replay{base: http.DefaultTransport, ttl: fetchTimeout}
	client := &http.Client{
		Timeout:       fetchTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		Transport:     transport,
	}
	keys := &keySet{issuer: issuer, transport: transport, ctx: oidc.ClientContext(context.Background(), client)}
	return &Verifier{
		keys: keys,
		verifier: oidc.NewVerifier(issuer, keys, &oidc.Config{
			ClientID:             audience,
			SupportedSigningAlgs: signingAlgorithms,
		}),
	}
}

// Verify returns the caller after checking signature, claims and access-token type.
func (v *Verifier) Verify(ctx context.Context, rawToken string) (*identity.Principal, error) {
	if rawToken == "" {
		return nil, identity.ErrUnauthenticated
	}
	var fetchErr error
	token, err := v.verifier.Verify(context.WithValue(ctx, fetchErrorKey{}, &fetchErr), rawToken)
	if err != nil {
		if fetchErr != nil {
			return nil, fmt.Errorf("%w: %w", identity.ErrUnavailable, fetchErr)
		}
		return nil, fmt.Errorf("%w: %w", identity.ErrUnauthenticated, err)
	}
	var claims struct {
		Type     json.RawMessage `json:"typ"`
		Username string          `json:"preferred_username"`
		Email    string          `json:"email"`
		Name     string          `json:"name"`
		Realm    struct {
			Roles []string `json:"roles"`
		} `json:"realm_access"`
	}
	if err := token.Claims(&claims); err != nil {
		return nil, fmt.Errorf("%w: malformed access-token claims", identity.ErrUnauthenticated)
	}
	if token.Subject == "" || !accessTokenType(rawToken, claims.Type) {
		return nil, fmt.Errorf("%w: missing subject or invalid access-token type", identity.ErrUnauthenticated)
	}
	return &identity.Principal{
		Subject: token.Subject, Username: claims.Username,
		Email: claims.Email, Name: claims.Name, Roles: claims.Realm.Roles,
	}, nil
}

func accessTokenType(rawToken string, claim json.RawMessage) bool {
	if len(claim) != 0 {
		var value string
		return json.Unmarshal(claim, &value) == nil && value == "Bearer"
	}
	header, _, _ := strings.Cut(rawToken, ".")
	data, err := base64.RawURLEncoding.DecodeString(header)
	if err != nil {
		return false
	}
	var parsed struct {
		Type string `json:"typ"`
	}
	return json.Unmarshal(data, &parsed) == nil &&
		(parsed.Type == "at+jwt" || parsed.Type == "application/at+jwt")
}

// Ready loads discovery and signing keys once; subsequent calls need no provider I/O.
func (v *Verifier) Ready(ctx context.Context) error {
	if v.keys.ready.Load() {
		return nil
	}
	var fetchErr error
	ctx = context.WithValue(ctx, fetchErrorKey{}, &fetchErr)
	// the deliberately invalid signature primes go-oidc's key cache
	_, _ = v.keys.VerifySignature(ctx, "eyJhbGciOiJSUzI1NiIsImtpZCI6Ii0ifQ.e30.AA")
	return fetchErr
}

var signingAlgorithms = []string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512", "PS256", "PS384", "PS512", "EdDSA"}

// go-oidc treats a successfully fetched empty set as a signature mismatch.
func checkSigningKeys(body []byte) error {
	var set struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := json.Unmarshal(body, &set); err != nil {
		return err
	}
	for _, raw := range set.Keys {
		var key jose.JSONWebKey
		if json.Unmarshal(raw, &key) == nil && key.Valid() && key.IsPublic() && (key.Use == "" || key.Use == "sig") && (key.Algorithm == "" || slices.Contains(signingAlgorithms, key.Algorithm)) {
			return nil
		}
	}
	return errors.New("issuer returned no usable signing keys")
}

type fetchErrorKey struct{}

type keySet struct {
	issuer    string
	ctx       context.Context
	discovery singleflight.Group
	remote    atomic.Pointer[oidc.RemoteKeySet]
	transport *replay
	ready     atomic.Bool
}

func (k *keySet) discover(ctx context.Context) (*oidc.RemoteKeySet, error) {
	if remote := k.remote.Load(); remote != nil {
		return remote, nil
	}
	result := k.discovery.DoChan("discovery", func() (any, error) {
		if remote := k.remote.Load(); remote != nil {
			return remote, nil
		}
		return k.loadDiscovery()
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case value := <-result:
		if value.Err != nil {
			return nil, value.Err
		}
		return value.Val.(*oidc.RemoteKeySet), nil
	}
}

func (k *keySet) loadDiscovery() (*oidc.RemoteKeySet, error) {
	provider, err := oidc.NewProvider(k.ctx, k.issuer)
	if err != nil {
		var mismatch *oidc.IssuerMismatchError
		if errors.As(err, &mismatch) {
			return nil, fmt.Errorf("%w: %w", ErrIssuerMismatch, err)
		}
		return nil, err
	}
	var metadata struct {
		JWKSURL string `json:"jwks_uri"`
	}
	if err := provider.Claims(&metadata); err != nil {
		return nil, err
	}
	endpoint, err := url.Parse(metadata.JWKSURL)
	if err != nil || endpoint.Hostname() == "" || (endpoint.Scheme != "https" && endpoint.Scheme != "http") || endpoint.User != nil || endpoint.Fragment != "" {
		return nil, errors.New("issuer returned an invalid jwks_uri")
	}
	k.transport.mu.Lock()
	k.transport.keysURL = metadata.JWKSURL
	k.transport.mu.Unlock()
	remote := oidc.NewRemoteKeySet(k.ctx, metadata.JWKSURL)
	k.remote.Store(remote)
	return remote, nil
}

func (k *keySet) VerifySignature(ctx context.Context, token string) ([]byte, error) {
	remote, err := k.discover(ctx)
	if err != nil {
		k.recordFetchError(ctx, err)
		return nil, err
	}
	payload, err := remote.VerifySignature(ctx, token)
	// go-oidc wraps fetch errors; signature failures have no underlying error
	if err != nil && errors.Unwrap(err) != nil {
		k.recordFetchError(ctx, err)
	} else {
		k.ready.Store(true)
	}
	return payload, err
}

func (*keySet) recordFetchError(ctx context.Context, err error) {
	if target, ok := ctx.Value(fetchErrorKey{}).(*error); ok {
		*target = err
	}
}
