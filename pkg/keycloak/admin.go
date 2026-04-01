package keycloak

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// AdminConfig holds credentials for the Keycloak Admin REST API service account.
type AdminConfig struct {
	BaseURL      string        // e.g. "http://localhost:8080"
	Realm        string        // e.g. "myrealm"
	ClientID     string        // service account client ID
	ClientSecret string        // service account client secret
	HTTPTimeout  time.Duration
}

// AdminUser represents a user as returned by the Keycloak Admin REST API.
type AdminUser struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Enabled   bool   `json:"enabled"`
}

type cachedAdminToken struct {
	accessToken string
	expiresAt   time.Time
}

// AdminClient calls Keycloak Admin REST API using a service account token.
type AdminClient struct {
	cfg    AdminConfig
	client *http.Client
	mu     sync.Mutex
	token  cachedAdminToken
}

// NewAdminClient creates a new AdminClient. Returns an error if required config is missing.
func NewAdminClient(cfg AdminConfig) (*AdminClient, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("keycloak admin: base URL is required")
	}
	if strings.TrimSpace(cfg.Realm) == "" {
		return nil, fmt.Errorf("keycloak admin: realm is required")
	}
	if strings.TrimSpace(cfg.ClientID) == "" {
		return nil, fmt.Errorf("keycloak admin: client ID is required")
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 10 * time.Second
	}
	return &AdminClient{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.HTTPTimeout},
	}, nil
}

func (a *AdminClient) tokenURL() string {
	base := strings.TrimRight(a.cfg.BaseURL, "/")
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", base, a.cfg.Realm)
}

func (a *AdminClient) adminURL(path string) string {
	base := strings.TrimRight(a.cfg.BaseURL, "/")
	return fmt.Sprintf("%s/admin/realms/%s%s", base, a.cfg.Realm, path)
}

// getToken returns a cached admin token, fetching a new one via client credentials if needed.
func (a *AdminClient) getToken(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.token.accessToken != "" && time.Now().Before(a.token.expiresAt) {
		return a.token.accessToken, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", a.cfg.ClientID)
	form.Set("client_secret", a.cfg.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.tokenURL(), strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("keycloak admin: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("keycloak admin: token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("keycloak admin: token request returned %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("keycloak admin: decode token response: %w", err)
	}

	expiry := time.Duration(tokenResp.ExpiresIn)*time.Second - 30*time.Second
	if expiry < 0 {
		expiry = 0
	}
	a.token = cachedAdminToken{
		accessToken: tokenResp.AccessToken,
		expiresAt:   time.Now().Add(expiry),
	}

	return a.token.accessToken, nil
}

func (a *AdminClient) doGet(ctx context.Context, rawURL string, out any) error {
	token, err := a.getToken(ctx)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("keycloak admin: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("keycloak admin: request to %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("keycloak admin: %s returned %d", rawURL, resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

// ListUsers returns up to 1000 users from Keycloak.
func (a *AdminClient) ListUsers(ctx context.Context) ([]AdminUser, error) {
	var users []AdminUser
	if err := a.doGet(ctx, a.adminURL("/users?max=1000"), &users); err != nil {
		return nil, err
	}
	return users, nil
}

// ListOrganizationMembers returns members of the organization identified by alias.
// Requires Keycloak 24+ with Organizations feature enabled.
func (a *AdminClient) ListOrganizationMembers(ctx context.Context, orgAlias string) ([]AdminUser, error) {
	searchURL := a.adminURL("/organizations?search=" + url.QueryEscape(orgAlias))

	var orgs []struct {
		ID    string `json:"id"`
		Alias string `json:"alias"`
		Name  string `json:"name"`
	}
	if err := a.doGet(ctx, searchURL, &orgs); err != nil {
		return nil, fmt.Errorf("keycloak admin: find organization %q: %w", orgAlias, err)
	}

	// Find exact alias match.
	orgID := ""
	for _, org := range orgs {
		if org.Alias == orgAlias {
			orgID = org.ID
			break
		}
	}
	if orgID == "" {
		return nil, fmt.Errorf("keycloak admin: organization %q not found", orgAlias)
	}

	var members []AdminUser
	if err := a.doGet(ctx, a.adminURL("/organizations/"+orgID+"/members"), &members); err != nil {
		return nil, err
	}
	return members, nil
}
