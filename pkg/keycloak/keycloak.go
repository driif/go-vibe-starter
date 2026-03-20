package keycloak

import (
	"context"
	"crypto"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	IssuerURL   string
	Audience    string
	HTTPTimeout time.Duration
	ClockSkew   time.Duration
}

type Verifier struct {
	cfg    Config
	client *http.Client
	cache  cacheState
}

func New(config Config) (*Verifier, error) {
	if strings.TrimSpace(config.IssuerURL) == "" {
		return nil, fmt.Errorf("%w: issuer url is required", ErrInvalidConfiguration)
	}
	if strings.TrimSpace(config.Audience) == "" {
		return nil, fmt.Errorf("%w: audience is required", ErrInvalidConfiguration)
	}
	if config.HTTPTimeout <= 0 {
		config.HTTPTimeout = 5 * time.Second
	}
	if config.ClockSkew < 0 {
		return nil, fmt.Errorf("%w: clock skew must be non-negative", ErrInvalidConfiguration)
	}
	if config.ClockSkew == 0 {
		config.ClockSkew = 30 * time.Second
	}

	return &Verifier{
		cfg: config,
		client: &http.Client{
			Timeout: config.HTTPTimeout,
		},
	}, nil
}

func (v *Verifier) Verify(ctx context.Context, rawToken string) (*Principal, error) {
	header, claims, rawClaims, parts, err := parseToken(rawToken)
	if err != nil {
		return nil, err
	}

	if header.KeyID == "" {
		return nil, ErrMalformedToken
	}

	hash, err := signingHash(header.Algorithm)
	if err != nil {
		return nil, err
	}

	if err := claims.validate(v.cfg, time.Now()); err != nil {
		return nil, err
	}

	publicKey, err := v.publicKey(ctx, header.KeyID)
	if err != nil {
		return nil, err
	}

	if err := verifySignature(parts[0]+"."+parts[1], parts[2], publicKey, hash); err != nil {
		return nil, err
	}

	return mapPrincipal(rawToken, claims, rawClaims), nil
}

func (v *Verifier) discoveryURL() string {
	return strings.TrimRight(v.cfg.IssuerURL, "/") + "/.well-known/openid-configuration"
}

func signingHash(alg string) (crypto.Hash, error) {
	switch alg {
	case "RS256":
		return crypto.SHA256, nil
	case "RS384":
		return crypto.SHA384, nil
	case "RS512":
		return crypto.SHA512, nil
	default:
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedSigningAlg, alg)
	}
}

func verifySignature(signingInput string, signatureSegment string, publicKey *rsa.PublicKey, hash crypto.Hash) error {
	signature, err := decodeSegment(signatureSegment)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMalformedToken, err)
	}

	hasher := hash.New()
	if _, err := hasher.Write([]byte(signingInput)); err != nil {
		return err
	}

	if err := rsa.VerifyPKCS1v15(publicKey, hash, hasher.Sum(nil), signature); err != nil {
		return ErrInvalidSignature
	}

	return nil
}

func decodeSegment(segment string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(segment)
}
