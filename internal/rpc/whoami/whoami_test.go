package whoami_test

import (
	"net/http"
	"testing"

	"connectrpc.com/connect/v2"
	"github.com/stretchr/testify/require"

	apiv1 "github.com/driif/go-vibe-starter/internal/gen/api/v1"
	"github.com/driif/go-vibe-starter/internal/gen/api/v1/apiv1connect"
	"github.com/driif/go-vibe-starter/internal/test"
)

func TestWhoAmI(t *testing.T) {
	t.Parallel()
	s := test.NewServer(t)
	resp, err := http.Get(s.URL + "/readyz")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	for _, roles := range [][]string{{"admin", "user"}, nil} {
		client := apiv1connect.NewWhoAmIServiceClient(s.Client(s.Token("alice", roles...)))
		got, err := client.WhoAmI(t.Context(), &apiv1.WhoAmIRequest{})
		require.NoError(t, err)
		require.Equal(t, "alice", got.GetSubject())
		require.Equal(t, "alice", got.GetUsername())
		require.Equal(t, "alice@example.test", got.GetEmail())
		require.Equal(t, "Test alice", got.GetName())
		require.ElementsMatch(t, roles, got.GetRoles())
	}
	client := apiv1connect.NewWhoAmIServiceClient(s.Client(""))
	_, err = client.WhoAmI(t.Context(), &apiv1.WhoAmIRequest{})
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
