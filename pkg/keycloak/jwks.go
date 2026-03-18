package keycloak

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/cubular-io/smartorder-gateway/internal/config"
	"github.com/golang-jwt/jwt"
)

type Certs struct {
	Keys      []Key `json:"keys"`
	ExpiresAt int64 `json:"expires_at"`
	Issuer    string
}

type Key struct {
	Kid      string   `json:"kid"`
	Kty      string   `json:"kty"`
	Alg      string   `json:"alg"`
	Use      string   `json:"use"`
	N        string   `json:"n"`
	E        string   `json:"e"`
	X5c      []string `json:"x5c"`
	X5t      string   `json:"x5t"`
	X5t_S256 string   `json:"x5t#S256"`
}

func GetPublicKeys(cfg config.Keycloak) (*Certs, error) {
	realmUrl := cfg.URL + "/realms/" + cfg.Realm
	url := realmUrl + "/protocol/openid-connect/certs"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		resp.Body.Close()
		return &Certs{}, err
	}
	defer resp.Body.Close()

	var certs *Certs
	err = json.NewDecoder(resp.Body).Decode(&certs)
	if err != nil {
		return certs, err
	}

	// Set expiration time in 5 minutes
	certs.ExpiresAt = time.Now().Add(5 * time.Minute).Unix()
	certs.Issuer = cfg.ISS + "/realms/" + cfg.Realm

	return certs, nil
}

// Expired checks if the certs are expired
func (c *Certs) Expired() bool {
	return time.Now().Unix() > c.ExpiresAt
}

// Refresh refreshes the certs
func (c *Certs) Refresh(cfg config.Keycloak) error {
	certs, err := GetPublicKeys(cfg)
	if err != nil {
		return err
	}

	*c = *certs

	return nil
}

// VerifyToken verifies the token
func (c *Certs) VerifyToken(tokenString string) (*Auth, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Find the correct key
		var key *Key
		for _, k := range c.Keys {
			if k.Kid == token.Header["kid"] {
				key = &k
				break
			}
		}
		if key == nil {
			return nil, fmt.Errorf("unable to find appropriate key")
		}

		// Convert key to RSA Public Key
		rsaPublicKey, err := jwkToRsaPublicKey(key)
		if err != nil {
			return nil, err
		}

		return rsaPublicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("Token is invalid")
	}

	auth := &Auth{}
	// Map JWT claims to Auth object
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		// Check issuer
		if claims["iss"] != c.Issuer {
			return nil, fmt.Errorf("invalid issuer")
		}

		auth = AuthFromJWT(claims)
	}

	// if err := auth.Validate(); err != nil {
	// 	return nil, err
	// }

	return auth, nil
}

func jwkToRsaPublicKey(key *Key) (*rsa.PublicKey, error) {
	decodedE, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, err
	}

	decodedN, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, err
	}

	e := new(big.Int).SetBytes(decodedE).Int64()
	n := new(big.Int).SetBytes(decodedN)

	return &rsa.PublicKey{
		N: n,
		E: int(e),
	}, nil
}
