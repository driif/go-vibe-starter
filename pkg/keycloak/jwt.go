package keycloak

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

type Principal struct {
	Subject       string
	Username      string
	Email         string
	EmailVerified bool
	Name          string
	GivenName     string
	FamilyName    string
	Organizations []string
	RealmRoles    []string
	ClientRoles   map[string][]string
	Claims        map[string]any
	Token         string
	Scopes        []string
}

func (p *Principal) HasRealmRole(role string) bool {
	return slices.Contains(p.RealmRoles, role)
}

func (p *Principal) HasClientRole(clientID, role string) bool {
	roles := p.ClientRoles[clientID]
	return slices.Contains(roles, role)
}

func (p *Principal) HasOrganization(org string) bool {
	return slices.Contains(p.Organizations, org)
}

func (p *Principal) HasScope(scope string) bool {
	return slices.Contains(p.Scopes, scope)
}

type tokenHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	Type      string `json:"typ"`
}

type rawTokenClaims struct {
	Issuer            string                    `json:"iss"`
	Subject           string                    `json:"sub"`
	Audience          audienceClaim             `json:"aud"`
	Expiry            int64                     `json:"exp"`
	IssuedAt          int64                     `json:"iat"`
	NotBefore         int64                     `json:"nbf"`
	PreferredUsername string                    `json:"preferred_username"`
	Email             string                    `json:"email"`
	EmailVerified     bool                      `json:"email_verified"`
	Name              string                    `json:"name"`
	GivenName         string                    `json:"given_name"`
	FamilyName        string                    `json:"family_name"`
	Organizations     organizationsClaim        `json:"organization"`
	RealmAccess       realmAccessClaim          `json:"realm_access"`
	ResourceAccess    map[string]resourceAccess `json:"resource_access"`
	Scope             string                    `json:"scope"`
}

type realmAccessClaim struct {
	Roles []string `json:"roles"`
}

type resourceAccess struct {
	Roles []string `json:"roles"`
}

type audienceClaim []string

func (a *audienceClaim) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*a = nil
		return nil
	}

	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		*a = []string{one}
		return nil
	}

	var many []string
	if err := json.Unmarshal(data, &many); err == nil {
		*a = many
		return nil
	}

	return fmt.Errorf("%w: invalid audience claim", ErrMalformedToken)
}

type organizationsClaim []string

func (o *organizationsClaim) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = nil
		return nil
	}

	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		if one == "" {
			*o = nil
			return nil
		}
		*o = []string{one}
		return nil
	}

	var many []string
	if err := json.Unmarshal(data, &many); err == nil {
		*o = many
		return nil
	}

	return fmt.Errorf("%w: invalid organization claim", ErrMalformedToken)
}

func parseToken(rawToken string) (tokenHeader, rawTokenClaims, map[string]any, []string, error) {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return tokenHeader{}, rawTokenClaims{}, nil, nil, ErrMalformedToken
	}

	headerBytes, err := decodeSegment(parts[0])
	if err != nil {
		return tokenHeader{}, rawTokenClaims{}, nil, nil, fmt.Errorf("%w: %v", ErrMalformedToken, err)
	}

	payloadBytes, err := decodeSegment(parts[1])
	if err != nil {
		return tokenHeader{}, rawTokenClaims{}, nil, nil, fmt.Errorf("%w: %v", ErrMalformedToken, err)
	}

	var header tokenHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return tokenHeader{}, rawTokenClaims{}, nil, nil, fmt.Errorf("%w: %v", ErrMalformedToken, err)
	}

	var claims rawTokenClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return tokenHeader{}, rawTokenClaims{}, nil, nil, fmt.Errorf("%w: %v", ErrMalformedToken, err)
	}

	rawClaims := make(map[string]any)
	if err := json.Unmarshal(payloadBytes, &rawClaims); err != nil {
		return tokenHeader{}, rawTokenClaims{}, nil, nil, fmt.Errorf("%w: %v", ErrMalformedToken, err)
	}

	return header, claims, rawClaims, parts, nil
}

func (c rawTokenClaims) validate(cfg Config, now time.Time) error {
	if c.Issuer != cfg.IssuerURL {
		return fmt.Errorf("%w: expected %q got %q", ErrIssuerMismatch, cfg.IssuerURL, c.Issuer)
	}

	if cfg.Audience != "" && !slices.Contains([]string(c.Audience), cfg.Audience) {
		return fmt.Errorf("%w: expected %q", ErrAudienceMismatch, cfg.Audience)
	}

	if c.Expiry == 0 {
		return ErrMalformedToken
	}

	if now.After(time.Unix(c.Expiry, 0).Add(cfg.ClockSkew)) {
		return ErrTokenExpired
	}

	if c.NotBefore > 0 && now.Add(cfg.ClockSkew).Before(time.Unix(c.NotBefore, 0)) {
		return ErrTokenExpired
	}

	return nil
}

func mapPrincipal(rawToken string, claims rawTokenClaims, raw map[string]any) *Principal {
	clientRoles := make(map[string][]string, len(claims.ResourceAccess))
	for clientID, access := range claims.ResourceAccess {
		if len(access.Roles) == 0 {
			continue
		}
		clientRoles[clientID] = append([]string(nil), access.Roles...)
	}

	return &Principal{
		Subject:       claims.Subject,
		Username:      claims.PreferredUsername,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
		GivenName:     claims.GivenName,
		FamilyName:    claims.FamilyName,
		Organizations: append([]string(nil), []string(claims.Organizations)...),
		RealmRoles:    append([]string(nil), claims.RealmAccess.Roles...),
		ClientRoles:   clientRoles,
		Claims:        raw,
		Token:         rawToken,
		Scopes:        strings.Fields(claims.Scope),
	}
}
