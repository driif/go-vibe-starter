package server

import (
	"net/http"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/stretchr/testify/require"

	apiv1 "github.com/driif/go-vibe-starter/internal/gen/api/v1"
	"github.com/driif/go-vibe-starter/internal/gen/api/v1/apiv1connect"
)

func TestErrors(t *testing.T) {
	srv := testHTTPServer(t)
	for _, protocol := range []struct {
		name    string
		options []connecthttp.Option
	}{
		{"connect", nil}, {"json", []connecthttp.Option{connecthttp.WithProtoJSON()}},
		{"grpc", []connecthttp.Option{connecthttp.WithGRPC()}}, {"grpc-web", []connecthttp.Option{connecthttp.WithGRPCWeb()}},
	} {
		t.Run(protocol.name, func(t *testing.T) {
			httpClient := srv.Client()
			if protocol.name == "grpc" {
				protocols := new(http.Protocols)
				protocols.SetUnencryptedHTTP2(true)
				transport := &http.Transport{Protocols: protocols}
				t.Cleanup(transport.CloseIdleConnections)
				httpClient = &http.Client{Transport: transport}
			}
			client := connect.NewClient(connecthttp.NewTransport(httpClient, srv.URL, protocol.options...))
			for _, tt := range []struct {
				mode    string
				code    connect.Code
				message string
			}{
				{"domain", connect.CodeNotFound, "test not found"},
				{"local", connect.CodeInvalidArgument, "invalid input"},
				{"plain", connect.CodeInternal, "internal error"},
				{"panic", connect.CodeInternal, "internal error"},
			} {
				t.Run(tt.mode, func(t *testing.T) {
					ctx, info := clientContext(t.Context(), "valid")
					info.RequestHeader().Set("X-Failure", tt.mode)
					var res apiv1.WhoAmIResponse
					err := client.CallUnary(ctx, failSpec, &apiv1.WhoAmIRequest{}, &res)
					require.Equal(t, tt.code, connect.CodeOf(err))
					var rpc *connect.Error
					require.ErrorAs(t, err, &rpc)
					require.Equal(t, tt.message, rpc.Message())
				})
			}
			who := apiv1connect.NewWhoAmIServiceClient(client)
			for _, tt := range []struct {
				token string
				code  connect.Code
			}{{"", connect.CodeUnauthenticated}, {"invalid", connect.CodeUnauthenticated}, {"down", connect.CodeUnavailable}} {
				ctx, _ := clientContext(t.Context(), tt.token)
				_, err := who.WhoAmI(ctx, &apiv1.WhoAmIRequest{})
				require.Equal(t, tt.code, connect.CodeOf(err))
				ctx, _ = clientContext(t.Context(), tt.token)
				stream, err := client.CallServerStream(ctx, tickSpec, &apiv1.WhoAmIRequest{})
				if err == nil {
					defer stream.Close()
					err = stream.Receive(&apiv1.WhoAmIResponse{})
				}
				require.Equal(t, tt.code, connect.CodeOf(err))
			}
		})
	}
}
