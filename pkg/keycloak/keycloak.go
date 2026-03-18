package keycloak

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/cubular-io/smartorder-gateway/internal/config"
)

type Keycloak struct {
	Certs  *Certs
	Config config.Keycloak
}

func NewWithConfigs(config config.Keycloak) *Keycloak {
	return &Keycloak{
		Certs:  nil,
		Config: config,
	}
}

// MakeRequestPayload makes URLEncoded payload from map
func MakeRequestPayload(m map[string]string) io.Reader {
	var payload string
	for k, v := range m {
		payload += k + "=" + v + "&"
	}
	// delete last &
	payload = strings.TrimSuffix(payload, "&")
	return strings.NewReader(payload)
}

// GetAuthFromResponse gets Auth from http response
func GetAuthFromResponse(res *http.Response) (Auth, error) {
	response, err := io.ReadAll(res.Body)
	if err != nil {
		return Auth{}, err
	}

	var keycloakAuth Auth
	// decode response
	err = json.Unmarshal(response, &keycloakAuth)
	if err != nil {
		return keycloakAuth, err
	}

	return keycloakAuth, nil
}

// GetAccessTokenFromPesponse gets Token from http response
func GetAccessTokenFromPesponse(res *http.Response) (Token, error) {

	response, err := io.ReadAll(res.Body)
	if err != nil {
		return Token{}, err
	}

	var keycloakToken Token
	// decode response
	err = json.Unmarshal(response, &keycloakToken)
	if err != nil {
		return keycloakToken, err
	}

	return keycloakToken, nil
}

// IntrospectPayload introspects payload with Keycloak
func IntrospectPayload(conf config.Keycloak, payload io.Reader) (*Auth, error) {
	url := conf.URL + "/realms/" + conf.Realm + "/protocol/openid-connect/token/introspect"
	res, err := http.Post(url, "application/x-www-form-urlencoded", payload)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	keycloakAuth, err := GetAuthFromResponse(res)
	if err != nil {
		return nil, err
	}

	return &keycloakAuth, nil
}

// IntrospectInternal introspects token for internal client
func IntrospectInternal(conf config.Keycloak, token string) (*Auth, error) {
	// Generate map for Keycloak Request
	m := make(map[string]string)
	m["client_id"] = conf.ClientID
	m["token"] = token
	m["client_secret"] = conf.ClientSecret

	// Make Payload for Keycloak Request from a map
	payload := MakeRequestPayload(m)

	return IntrospectPayload(conf, payload)
}

// IntrospectExternal introspects token for external client
func IntrospectExternal(conf config.Keycloak, token string) (*Auth, error) {
	// Generate map for Keycloak Request
	m := make(map[string]string)
	m["client_id"] = conf.ExternalClientID
	m["token"] = token
	m["client_secret"] = conf.ExternalClientSecret

	// Make Payload for Keycloak Request from a map
	payload := MakeRequestPayload(m)

	return IntrospectPayload(conf, payload)
}
