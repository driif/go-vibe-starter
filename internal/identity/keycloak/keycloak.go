// Package keycloak verifies access tokens with the issuer's discovered signing keys.
package keycloak

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	client := &http.Client{
		Timeout:   fetchTimeout,
		Transport: &replay{base: http.DefaultTransport, ttl: fetchTimeout},
	}
	keys := &keySet{issuer: issuer, ctx: oidc.ClientContext(context.Background(), client)}
	return &Verifier{
		keys: keys,
		verifier: oidc.NewVerifier(issuer, keys, &oidc.Config{
			ClientID: audience,
			SupportedSigningAlgs: []string{
				"RS256", "RS384", "RS512", "ES256", "ES384", "ES512",
				"PS256", "PS384", "PS512", "EdDSA",
			},
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
func (v *Verifier) Ready(context.Context) error {
	if v.keys.ready.Load() {
		return nil
	}
	var fetchErr error
	ctx := context.WithValue(v.keys.ctx, fetchErrorKey{}, &fetchErr)
	// the deliberately invalid signature primes go-oidc's key cache
	_, _ = v.keys.VerifySignature(ctx, "eyJhbGciOiJSUzI1NiIsImtpZCI6Ii0ifQ.e30.AA")
	return fetchErr
}

type fetchErrorKey struct{}

type keySet struct {
	issuer string
	ctx    context.Context
	mu     sync.Mutex
	remote *oidc.RemoteKeySet
	ready  atomic.Bool
}

func (k *keySet) discover() (*oidc.RemoteKeySet, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.remote != nil {
		return k.remote, nil
	}
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
	k.remote = oidc.NewRemoteKeySet(k.ctx, metadata.JWKSURL)
	return k.remote, nil
}

func (k *keySet) VerifySignature(ctx context.Context, token string) ([]byte, error) {
	remote, err := k.discover()
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
