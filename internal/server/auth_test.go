package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/stretchr/testify/require"

	apiv1 "github.com/driif/go-vibe-starter/internal/gen/api/v1"
	"github.com/driif/go-vibe-starter/internal/gen/api/v1/apiv1connect"
	"github.com/driif/go-vibe-starter/internal/identity"
)

type unreadableBody struct{}

func (unreadableBody) Read([]byte) (int, error) { panic("request body read before authentication") }
func (unreadableBody) Close() error             { return nil }

func TestAuth_AnswersBeforeBody(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api.v1.WhoAmIService/WhoAmI", nil)
	r.Body = unreadableBody{}
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testHandler(t).ServeHTTP(w, r)
	require.Equal(t, 401, w.Code)
}

func TestAuth_DevPrincipal(t *testing.T) {
	srv := testHTTPServer(t, func(d *Deps) { d.Verifier = identity.DevVerifier{Roles: []string{"user"}} })
	client := apiv1connect.NewWhoAmIServiceClient(connect.NewClient(connecthttp.NewTransport(srv.Client(), srv.URL)))
	res, err := client.WhoAmI(t.Context(), &apiv1.WhoAmIRequest{})
	require.NoError(t, err)
	require.Equal(t, "local-dev", res.GetSubject())
	require.Equal(t, []string{"user"}, res.GetRoles())
}

func TestAuth_Wire(t *testing.T) {
	for _, contentType := range []string{"application/proto", "application/json", "application/connect+proto", "application/grpc", "application/grpc-web+proto", "text/plain"} {
		for _, tt := range []struct {
			token, challenge string
			status           int
			grpcStatus       string
		}{
			{"", "Bearer", 401, "16"},
			{"bad", `Bearer error="invalid_token"`, 401, "16"},
			{"down", "", 503, "14"},
		} {
			t.Run(contentType+"/"+tt.token, func(t *testing.T) {
				r := httptest.NewRequest(http.MethodPost, "/api.v1.WhoAmIService/WhoAmI", nil)
				r.Header.Set("Content-Type", contentType)
				if tt.token != "" {
					r.Header.Set("Authorization", "Bearer "+tt.token)
				}
				w := httptest.NewRecorder()
				testHandler(t).ServeHTTP(w, r)
				wantStatus := tt.status
				if contentType == "application/connect+proto" || contentType == "application/grpc" || contentType == "application/grpc-web+proto" {
					wantStatus = 200
				}
				require.Equal(t, wantStatus, w.Code)
				require.Equal(t, tt.challenge, w.Header().Get("WWW-Authenticate"))
				if contentType == "application/grpc" {
					require.Equal(t, tt.grpcStatus, w.Result().Trailer.Get("Grpc-Status"))
				}
				if contentType == "application/grpc-web+proto" {
					require.Equal(t, tt.grpcStatus, w.Header().Get("Grpc-Status"))
				}
			})
		}
	}
}
