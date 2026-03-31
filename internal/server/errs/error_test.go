package errs

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	Write(rec, http.StatusUnauthorized, errors.New("auth: unauthenticated"))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}

	var got response
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.Status != http.StatusUnauthorized {
		t.Fatalf("expected status 401 in body, got %d", got.Status)
	}
	if got.Detail != "auth: unauthenticated" {
		t.Fatalf("unexpected detail: %s", got.Detail)
	}
}

func TestWriteValidation(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteValidation(rec, errors.New("validation failed"), []string{"name is required", "email is invalid"})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var got validationResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(got.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(got.Errors))
	}
}
