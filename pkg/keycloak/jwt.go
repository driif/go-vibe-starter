package keycloak

import (
	"fmt"
	"time"

	"github.com/cubular-io/smartorder-pkg/uuid"
	"github.com/golang-jwt/jwt"
)

// Auth represents a keycloak auth
type Auth struct {
	Exp               int    `json:"exp"`
	Iat               int    `json:"iat"`
	Jti               string `json:"jti"`
	Iss               string `json:"iss"`
	Aud               string `json:"aud"`
	Sub               string `json:"sub"`
	Typ               string `json:"typ"`
	Azp               string `json:"azp"`
	SessionState      string `json:"session_state"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	ResourceAccess struct {
		Account struct {
			Roles []string `json:"roles"`
		} `json:"account"`
	} `json:"resource_access"`
	Scope        string     `json:"scope"`
	Sid          string     `json:"sid"`
	Organization uuid.UUID  `json:"organization"`
	LocationID   *uuid.UUID `json:"location"`
	ID           uuid.UUID  `json:"uid"`
	Name         string     `json:"name"`
	LastName     string     `json:"family_name"`
	FirstName    string     `json:"given_name"`
	Active       bool       `json:"active"`
	AvatarURL    string     `json:"avatar"`
	Username     string     `json:"username"`
}

// Token represents a keycloak token
type Token struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	SessionState     string `json:"session_state"`
	Scope            string `json:"scope"`
}

func AuthFromJWT(claims jwt.MapClaims) *Auth {
	auth := &Auth{}

	if v, ok := claims["exp"].(float64); ok {
		auth.Exp = int(v)
	}
	if v, ok := claims["iat"].(float64); ok {
		auth.Iat = int(v)
	}
	if v, ok := claims["jti"].(string); ok {
		auth.Jti = v
	}
	if v, ok := claims["iss"].(string); ok {
		auth.Iss = v
	}
	if v, ok := claims["sub"].(string); ok {
		auth.Sub = v
	}
	if v, ok := claims["typ"].(string); ok {
		auth.Typ = v
	}
	if v, ok := claims["azp"].(string); ok {
		auth.Azp = v
	}
	if v, ok := claims["session_state"].(string); ok {
		auth.SessionState = v
	}
	if v, ok := claims["preferred_username"].(string); ok {
		auth.PreferredUsername = v
	}
	if v, ok := claims["email"].(string); ok {
		auth.Email = v
	}
	if v, ok := claims["email_verified"].(bool); ok {
		auth.EmailVerified = v
	}
	if v, ok := claims["scope"].(string); ok {
		auth.Scope = v
	}
	if v, ok := claims["sid"].(string); ok {
		auth.Sid = v
	}
	if v, ok := claims["organization"].(string); ok {
		if uuid, err := uuid.Parse(v); err == nil {
			auth.Organization = uuid
		}
	}
	if v, ok := claims["uid"].(string); ok {
		if uuid, err := uuid.Parse(v); err == nil {
			auth.ID = uuid
		}
	}

	if v, ok := claims["location"].(string); ok {
		if uuid, err := uuid.Parse(v); err == nil {
			auth.LocationID = &uuid
		}
	}

	if v, ok := claims["family_name"].(string); ok {
		auth.LastName = v
	}
	if v, ok := claims["given_name"].(string); ok {
		auth.FirstName = v
	}
	if v, ok := claims["name"].(string); ok {
		auth.Name = v
	}
	if v, ok := claims["active"].(bool); ok {
		auth.Active = v
	}
	if v, ok := claims["avatar"].(string); ok {
		auth.AvatarURL = v
	}
	if v, ok := claims["realm_access"].(map[string]interface{}); ok {
		if roles, ok := v["roles"].([]interface{}); ok {
			for _, role := range roles {
				if roleStr, ok := role.(string); ok {
					auth.RealmAccess.Roles = append(auth.RealmAccess.Roles, roleStr)
				}
			}
		}
	}

	return auth
}

type ValidationError []string

func (v ValidationError) Error() string {
	return fmt.Sprintf("Validation errors: %v", []string(v))
}

func (a *Auth) Validate() error {
	var err ValidationError

	switch {
	case a.ID == uuid.Nil:
		err = append(err, "invalid id")
	case a.Organization == uuid.Nil:
		err = append(err, "invalid organization")
	case a.Name == "":
		err = append(err, "invalid username")
	case a.Email == "":
		err = append(err, "invalid email")
	case a.FirstName == "":
		err = append(err, "invalid first name")
	case a.LastName == "":
		err = append(err, "invalid last name")
	case !a.Active:
		err = append(err, "invalid active")
	case a.Exp < int(time.Now().Unix()):
		err = append(err, "invalid expiration")
	}

	if len(err) == 0 {
		return nil
	}

	return err
}
