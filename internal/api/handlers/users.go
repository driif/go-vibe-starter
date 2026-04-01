package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/driif/go-vibe-starter/internal/server"
	"github.com/driif/go-vibe-starter/internal/server/auth"
	"github.com/driif/go-vibe-starter/internal/server/errs"
	"github.com/driif/go-vibe-starter/pkg/keycloak"
)

// userResponse is the unified user shape returned by all user endpoints.
type userResponse struct {
	ID            string   `json:"id"`
	Username      string   `json:"username"`
	Email         string   `json:"email,omitempty"`
	FirstName     string   `json:"firstName,omitempty"`
	LastName      string   `json:"lastName,omitempty"`
	Enabled       bool     `json:"enabled"`
	Organizations []string `json:"organizations,omitempty"`
}

func principalToResponse(p *keycloak.Principal) userResponse {
	return userResponse{
		ID:            p.Subject,
		Username:      p.Username,
		Email:         p.Email,
		FirstName:     p.GivenName,
		LastName:      p.FamilyName,
		Enabled:       true,
		Organizations: p.Organizations,
	}
}

func adminUserToResponse(u keycloak.AdminUser) userResponse {
	return userResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Enabled:   u.Enabled,
	}
}

// GetMe returns the current user's profile from JWT claims.
func GetMe(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		errs.Write(w, http.StatusUnauthorized, fmt.Errorf("missing principal"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(principalToResponse(principal))
}

// ListUsers returns users from Keycloak Admin API.
// Accepts optional ?organization= query param to filter by org alias.
func ListUsers(s *server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.KeycloakAdmin == nil {
			errs.Write(w, http.StatusServiceUnavailable, fmt.Errorf("keycloak admin service account not configured"))
			return
		}

		var adminUsers []keycloak.AdminUser
		var err error

		if org := r.URL.Query().Get("organization"); org != "" {
			adminUsers, err = s.KeycloakAdmin.ListOrganizationMembers(r.Context(), org)
		} else {
			adminUsers, err = s.KeycloakAdmin.ListUsers(r.Context())
		}

		if err != nil {
			errs.Write(w, http.StatusBadGateway, err)
			return
		}

		users := make([]userResponse, len(adminUsers))
		for i, u := range adminUsers {
			users[i] = adminUserToResponse(u)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}
