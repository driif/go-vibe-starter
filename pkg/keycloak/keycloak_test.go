package keycloak

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestVerify(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	modulus := base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes())
	exponent := base64.RawURLEncoding.EncodeToString(bigEndianBytes(privateKey.PublicKey.E))

	const issuerURL = "https://issuer.example/realms/test"

	verifier, err := New(Config{
		IssuerURL: issuerURL,
		Audience:  "api",
	})
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}
	verifier.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case issuerURL + "/.well-known/openid-configuration":
				return jsonResponse(map[string]any{
					"issuer":   issuerURL,
					"jwks_uri": issuerURL + "/jwks",
				}), nil
			case issuerURL + "/jwks":
				return jsonResponse(map[string]any{
					"keys": []map[string]any{{
						"kid": "test-kid",
						"kty": "RSA",
						"n":   modulus,
						"e":   exponent,
					}},
				}), nil
			default:
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("not found")),
					Header:     make(http.Header),
				}, nil
			}
		}),
	}

	token, err := signedToken(privateKey, issuerURL, "api", "test-kid", map[string]any{
		"preferred_username": "alice",
		"realm_access": map[string]any{
			"roles": []string{"admin"},
		},
		"resource_access": map[string]any{
			"api": map[string]any{
				"roles": []string{"write"},
			},
		},
		"organization": []string{"org-1"},
	})
	if err != nil {
		t.Fatalf("signed token: %v", err)
	}

	principal, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}

	if principal.Username != "alice" {
		t.Fatalf("expected username alice, got %q", principal.Username)
	}
	if !principal.HasRealmRole("admin") {
		t.Fatal("expected admin role")
	}
	if !principal.HasClientRole("api", "write") {
		t.Fatal("expected api/write role")
	}
	if !principal.HasOrganization("org-1") {
		t.Fatal("expected org-1 membership")
	}
}

func TestVerifyRejectsWrongAudience(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	modulus := base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes())
	exponent := base64.RawURLEncoding.EncodeToString(bigEndianBytes(privateKey.PublicKey.E))

	const issuerURL = "https://issuer.example/realms/test"

	verifier, err := New(Config{
		IssuerURL: issuerURL,
		Audience:  "api",
	})
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}
	verifier.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case issuerURL + "/.well-known/openid-configuration":
				return jsonResponse(map[string]any{
					"issuer":   issuerURL,
					"jwks_uri": issuerURL + "/jwks",
				}), nil
			case issuerURL + "/jwks":
				return jsonResponse(map[string]any{
					"keys": []map[string]any{{
						"kid": "test-kid",
						"kty": "RSA",
						"n":   modulus,
						"e":   exponent,
					}},
				}), nil
			default:
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("not found")),
					Header:     make(http.Header),
				}, nil
			}
		}),
	}

	token, err := signedToken(privateKey, issuerURL, "frontend", "test-kid", nil)
	if err != nil {
		t.Fatalf("signed token: %v", err)
	}

	if _, err := verifier.Verify(context.Background(), token); err == nil {
		t.Fatal("expected audience verification to fail")
	}
}

func signedToken(privateKey *rsa.PrivateKey, issuer string, audience string, keyID string, extra map[string]any) (string, error) {
	header := map[string]any{
		"alg": "RS256",
		"typ": "JWT",
		"kid": keyID,
	}

	now := time.Now()
	claims := map[string]any{
		"iss": issuer,
		"sub": "user-1",
		"aud": audience,
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
	}
	for key, value := range extra {
		claims[key] = value
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	sum := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}

	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func bigEndianBytes(value int) []byte {
	bytes := []byte{}
	for value > 0 {
		bytes = append([]byte{byte(value % 256)}, bytes...)
		value /= 256
	}
	return bytes
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func jsonResponse(value any) *http.Response {
	body, _ := json.Marshal(value)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(string(body))),
		Header: http.Header{
			"Content-Type":  []string{"application/json"},
			"Cache-Control": []string{"max-age=300"},
		},
	}
}

func TestPEMHelper(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(privateKey)
	block := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})
	if len(block) == 0 {
		t.Fatal("expected PEM block")
	}
}
