package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/driif/go-vibe-starter/pkg/keycloak"
)

type stubVerifier struct {
	principal *keycloak.Principal
	err       error
}

func (s stubVerifier) Verify(context.Context, string) (*keycloak.Principal, error) {
	return s.principal, s.err
}

func TestAuthenticateRequired(t *testing.T) {
	nextCalled := false
	handler := Authenticate(stubVerifier{
		principal: &keycloak.Principal{Subject: "user-1"},
	}, Options{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		if _, ok := PrincipalFromContext(r.Context()); !ok {
			t.Fatal("expected principal in context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
}

func TestAuthenticateRequiredAcceptsCaseInsensitiveBearerScheme(t *testing.T) {
	handler := Authenticate(stubVerifier{
		principal: &keycloak.Principal{Subject: "user-1"},
	}, Options{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestAuthenticateOptionalAllowsMissingToken(t *testing.T) {
	handler := Authenticate(stubVerifier{}, Options{Mode: ModeOptional})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestAuthenticateRejectsWrongScheme(t *testing.T) {
	handler := Authenticate(stubVerifier{
		principal: &keycloak.Principal{Subject: "user-1"},
	}, Options{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireRealmRoles(t *testing.T) {
	handler := RequireRealmRoles(true, "admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(withPrincipal(req.Context(), &keycloak.Principal{
		RealmRoles: []string{"admin"},
	}, "token"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}
