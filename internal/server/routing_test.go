package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRouting(t *testing.T) {
	h := testHandler(t)
	for _, tt := range []struct {
		name, method, path, token, contentType, body string
		status                                       int
	}{
		{"health", http.MethodGet, "/healthz", "", "", "", 200},
		{"unknown", http.MethodGet, "/absent", "", "", "", 404},
		{"wrong method", http.MethodPost, "/healthz", "", "", "", 405},
		{"rpc missing token", http.MethodPost, "/api.v1.WhoAmIService/WhoAmI", "", "application/json", "{}", 401},
		{"rpc", http.MethodPost, "/api.v1.WhoAmIService/WhoAmI", "valid", "application/json", "{}", 200},
		{"rpc GET", http.MethodGet, "/api.v1.WhoAmIService/WhoAmI?connect=v1&encoding=json&message=%7B%7D", "valid", "", "", 200},
		{"unknown rpc missing token", http.MethodPost, "/api.v1.WhoAmIService/Absent", "", "application/json", "{}", 401},
		{"unknown rpc", http.MethodPost, "/api.v1.WhoAmIService/Absent", "valid", "application/json", "{}", 501},
		{"wrong content type", http.MethodPost, "/api.v1.WhoAmIService/WhoAmI", "valid", "text/plain", "{}", 415},
		{"invalid json", http.MethodPost, "/api.v1.WhoAmIService/WhoAmI", "valid", "application/json", "{", 400},
		{"plain missing token", http.MethodGet, "/test/plain", "", "", "", 401},
		{"plain", http.MethodGet, "/test/plain", "valid", "", "", 200},
		{"panic", http.MethodGet, "/test/panic", "valid", "", "", 500},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.token != "" {
				r.Header.Set("Authorization", "Bearer "+tt.token)
			}
			r.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			require.Equal(t, tt.status, w.Code, w.Body.String())
			require.NotEmpty(t, w.Header().Get("X-Request-Id"))
			require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		})
	}
}

func TestHandler_RequiresDependencies(t *testing.T) {
	_, err := Handler(Deps{})
	require.Error(t, err)
}

func TestBodyLimit(t *testing.T) {
	h := testHandler(t)
	r := httptest.NewRequest(http.MethodPost, "/test/echo", strings.NewReader(strings.Repeat("x", maxRequestBytes+1)))
	r.Header.Set("Authorization", "Bearer valid")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, 413, w.Code)
}

func TestManagement(t *testing.T) {
	srv := testHTTPServer(t)
	for _, path := range []string{"/metrics", "/debug/pprof/"} {
		res, err := srv.Client().Get(srv.URL + path)
		require.NoError(t, err)
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
		require.Equal(t, 404, res.StatusCode)
	}
}
