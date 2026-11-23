package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunRefusesMissingConfiguration(t *testing.T) {
	t.Setenv("AUTH_PROVIDER", "")
	t.Setenv("DATABASE_URL", "")
	command := root()
	command.SetArgs([]string{"run"})
	err := command.ExecuteContext(t.Context())
	require.ErrorContains(t, err, "AUTH_PROVIDER is required")
	require.ErrorContains(t, err, "DATABASE_URL is required")
}

func TestProbe(t *testing.T) {
	for _, code := range []int{200, 503} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/readyz", r.URL.Path)
			w.WriteHeader(code)
		}))
		t.Cleanup(srv.Close)
		u, err := url.Parse(srv.URL)
		require.NoError(t, err)
		t.Setenv("LISTEN_ADDR", ":"+u.Port())
		command := root()
		command.SetArgs([]string{"probe"})
		err = command.ExecuteContext(t.Context())
		if code == 200 {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, "503")
		}
	}
}
