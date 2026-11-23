package server

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/stretchr/testify/require"

	apiv1 "github.com/driif/go-vibe-starter/internal/gen/api/v1"
)

func TestStreaming_Flushes(t *testing.T) {
	for _, h2 := range []bool{false, true} {
		name := "http1"
		if h2 {
			name = "h2c"
		}
		t.Run(name, func(t *testing.T) {
			release := make(chan struct{})
			srv := serveTestHandler(t, gatedHandler(t, release))
			httpClient := srv.Client()
			if h2 {
				protocols := new(http.Protocols)
				protocols.SetUnencryptedHTTP2(true)
				transport := &http.Transport{Protocols: protocols}
				t.Cleanup(transport.CloseIdleConnections)
				httpClient = &http.Client{Transport: transport}
			}
			client := connect.NewClient(connecthttp.NewTransport(httpClient, srv.URL))
			bounded, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			ctx, _ := clientContext(bounded, "valid")
			stream, err := client.CallServerStream(ctx, tickSpec, &apiv1.WhoAmIRequest{})
			require.NoError(t, err)
			defer stream.Close()
			var first, second apiv1.WhoAmIResponse
			require.NoError(t, stream.Receive(&first))
			require.Equal(t, "first", first.GetSubject())
			close(release)
			require.NoError(t, stream.Receive(&second))
			require.Equal(t, "second", second.GetSubject())
			require.ErrorIs(t, stream.Receive(&apiv1.WhoAmIResponse{}), io.EOF)
		})
	}
}

func TestClientStreaming(t *testing.T) {
	srv := testHTTPServer(t)
	client := connect.NewClient(connecthttp.NewTransport(srv.Client(), srv.URL))
	ctx, _ := clientContext(t.Context(), "valid")
	stream, err := client.CallClientStream(ctx, sumSpec)
	require.NoError(t, err)
	defer stream.Close()
	for range 3 {
		require.NoError(t, stream.Send(&apiv1.WhoAmIRequest{}))
	}
	require.NoError(t, stream.CloseSend())
	var res apiv1.WhoAmIResponse
	require.NoError(t, stream.Receive(&res))
	require.Len(t, res.GetRoles(), 3)
}
