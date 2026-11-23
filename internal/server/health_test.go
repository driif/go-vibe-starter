package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

type fakeReadier struct{ err error }

func (f fakeReadier) Ready(context.Context) error { return f.err }

func TestHealthz(t *testing.T) {
	w := httptest.NewRecorder()
	healthz(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, 200, w.Code)
	require.Empty(t, w.Body.String())
}

func TestReadyz(t *testing.T) {
	for _, tt := range []struct {
		name     string
		dbErr    error
		verifier any
		status   int
		body     string
	}{
		{"ready", nil, fakeReadier{}, 200, ""},
		{"database unavailable", errors.New("password=secret"), fakeReadier{}, 503, "database unavailable\n"},
		{"identity unavailable", nil, fakeReadier{errors.New("private issuer")}, 503, "identity provider unavailable\n"},
		{"no readiness interface", nil, struct{}{}, 200, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			readyz(fakePinger{tt.dbErr}, tt.verifier).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			require.Equal(t, tt.status, w.Code)
			require.Equal(t, tt.body, w.Body.String())
		})
	}
}
