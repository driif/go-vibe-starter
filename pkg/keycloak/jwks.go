package keycloak

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultCacheTTL = 5 * time.Minute

type discoveryDocument struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

type jwksDocument struct {
	Keys []jsonWebKey `json:"keys"`
}

type jsonWebKey struct {
	KeyID string `json:"kid"`
	Type  string `json:"kty"`
	N     string `json:"n"`
	E     string `json:"e"`
}

type cacheEntry[T any] struct {
	Value     T
	ExpiresAt time.Time
}

type cacheState struct {
	mu        sync.RWMutex
	discovery cacheEntry[discoveryDocument]
	keys      cacheEntry[map[string]*rsa.PublicKey]
}

func (s *cacheState) discoveryValue() (discoveryDocument, time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.discovery.Value.JWKSURI == "" {
		return discoveryDocument{}, time.Time{}, false
	}
	return s.discovery.Value, s.discovery.ExpiresAt, true
}

func (s *cacheState) setDiscovery(doc discoveryDocument, expiry time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.discovery = cacheEntry[discoveryDocument]{Value: doc, ExpiresAt: expiry}
}

func (s *cacheState) keyValue(keyID string) (*rsa.PublicKey, time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.keys.Value) == 0 {
		return nil, time.Time{}, false
	}
	key, ok := s.keys.Value[keyID]
	return key, s.keys.ExpiresAt, ok
}

func (s *cacheState) setKeys(keys map[string]*rsa.PublicKey, expiry time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys = cacheEntry[map[string]*rsa.PublicKey]{Value: keys, ExpiresAt: expiry}
}

func (v *Verifier) discovery(ctx context.Context, force bool) (discoveryDocument, error) {
	if !force {
		doc, expiry, ok := v.cache.discoveryValue()
		if ok && time.Now().Before(expiry) {
			return doc, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.discoveryURL(), nil)
	if err != nil {
		return discoveryDocument{}, err
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return discoveryDocument{}, fmt.Errorf("%w: %v", ErrDiscoveryFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return discoveryDocument{}, fmt.Errorf("%w: %s", ErrDiscoveryFailed, strings.TrimSpace(string(body)))
	}

	var doc discoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return discoveryDocument{}, fmt.Errorf("%w: %v", ErrDiscoveryFailed, err)
	}

	expiry := responseExpiry(resp.Header, defaultCacheTTL)
	v.cache.setDiscovery(doc, expiry)

	return doc, nil
}

func (v *Verifier) publicKey(ctx context.Context, keyID string) (*rsa.PublicKey, error) {
	key, expiry, ok := v.cache.keyValue(keyID)
	if ok && time.Now().Before(expiry) {
		return key, nil
	}

	if err := v.refreshKeys(ctx, false); err != nil {
		return nil, err
	}

	key, _, ok = v.cache.keyValue(keyID)
	if ok {
		return key, nil
	}

	if err := v.refreshKeys(ctx, true); err != nil {
		return nil, err
	}

	key, _, ok = v.cache.keyValue(keyID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrKeyNotFound, keyID)
	}

	return key, nil
}

func (v *Verifier) refreshKeys(ctx context.Context, forceDiscovery bool) error {
	doc, err := v.discovery(ctx, forceDiscovery)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, doc.JWKSURI, nil)
	if err != nil {
		return err
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrJWKSFetchFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%w: %s", ErrJWKSFetchFailed, strings.TrimSpace(string(body)))
	}

	var jwks jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("%w: %v", ErrJWKSFetchFailed, err)
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, jwk := range jwks.Keys {
		if jwk.Type != "RSA" || jwk.KeyID == "" {
			continue
		}

		key, err := jwk.rsaPublicKey()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrJWKSFetchFailed, err)
		}
		keys[jwk.KeyID] = key
	}

	v.cache.setKeys(keys, responseExpiry(resp.Header, defaultCacheTTL))
	return nil
}

func (jwk jsonWebKey) rsaPublicKey() (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, err
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())
	if e == 0 {
		return nil, fmt.Errorf("invalid exponent")
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}

func responseExpiry(headers http.Header, fallback time.Duration) time.Time {
	cacheControl := headers.Get("Cache-Control")
	if cacheControl != "" {
		for _, part := range strings.Split(cacheControl, ",") {
			part = strings.TrimSpace(part)
			if !strings.HasPrefix(part, "max-age=") {
				continue
			}
			seconds, err := strconv.Atoi(strings.TrimPrefix(part, "max-age="))
			if err == nil {
				return time.Now().Add(time.Duration(seconds) * time.Second)
			}
		}
	}

	if expires := headers.Get("Expires"); expires != "" {
		if expiry, err := http.ParseTime(expires); err == nil {
			return expiry
		}
	}

	return time.Now().Add(fallback)
}
