package auth

import "errors"

var (
	ErrMissingBearerToken = errors.New("auth: missing bearer token")
	ErrMalformedToken     = errors.New("auth: malformed bearer token")
	ErrUnauthenticated    = errors.New("auth: unauthenticated")
	ErrForbidden          = errors.New("auth: forbidden")
)
