package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDevPrincipalInjectsWithoutHeader(t *testing.T) {
	handler := DevPrincipal("local-dev", []string{"admin"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok {
			t.Fatal("expected principal in context")
		}
		if principal.Subject != "local-dev" {
			t.Fatalf("unexpected subject %q", principal.Subject)
		}
		if !principal.HasRealmRole("admin") {
			t.Fatal("expected admin realm role")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestDevPrincipalIgnoresAuthorizationHeader(t *testing.T) {
	handler := DevPrincipal("local-dev", nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}
