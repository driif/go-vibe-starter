package keycloak

import "errors"

var (
	ErrMalformedToken        = errors.New("keycloak: malformed token")
	ErrUnsupportedSigningAlg = errors.New("keycloak: unsupported signing algorithm")
	ErrInvalidSignature      = errors.New("keycloak: invalid token signature")
	ErrKeyNotFound           = errors.New("keycloak: signing key not found")
	ErrTokenExpired          = errors.New("keycloak: token expired")
	ErrIssuerMismatch        = errors.New("keycloak: issuer mismatch")
	ErrAudienceMismatch      = errors.New("keycloak: audience mismatch")
	ErrDiscoveryFailed       = errors.New("keycloak: discovery failed")
	ErrJWKSFetchFailed       = errors.New("keycloak: jwks fetch failed")
	ErrInvalidConfiguration  = errors.New("keycloak: invalid configuration")
)
