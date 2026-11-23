package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect/v2"
	"github.com/go-pkgz/routegroup"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/driif/go-vibe-starter/internal/apperr"
	apiv1 "github.com/driif/go-vibe-starter/internal/gen/api/v1"
	"github.com/driif/go-vibe-starter/internal/gen/api/v1/apiv1connect"
	"github.com/driif/go-vibe-starter/internal/identity"
	"github.com/driif/go-vibe-starter/internal/rpc/whoami"
	"github.com/driif/go-vibe-starter/internal/server/middleware"
)

var failSpec = connect.Spec{Procedure: "/test.v1.TestService/Fail", StreamType: connect.StreamTypeUnary}
var tickSpec = connect.Spec{Procedure: "/test.v1.TestService/Tick", StreamType: connect.StreamTypeServer}
var sumSpec = connect.Spec{Procedure: "/test.v1.TestService/Sum", StreamType: connect.StreamTypeClient}

type tokenVerifier struct{}

func (tokenVerifier) Verify(_ context.Context, token string) (*identity.Principal, error) {
	if token == "down" {
		return nil, identity.ErrUnavailable
	}
	if token != "valid" {
		return nil, identity.ErrUnauthenticated
	}
	return &identity.Principal{Subject: "alice", Username: "alice", Roles: []string{"user"}}, nil
}

func testHandler(t *testing.T, change ...func(*Deps)) http.Handler {
	t.Helper()
	d := Deps{DB: new(pgxpool.Pool), Verifier: tokenVerifier{}, CORSOrigins: []string{"https://app.example"}}
	for _, fn := range change {
		fn(&d)
	}
	h, err := build(d, func(rpc *connect.Server, plain *routegroup.Bundle, _ Deps) {
		apiv1connect.RegisterWhoAmIServiceHandler(rpc, whoami.Handler{})
		rpc.Register(connect.Method{Spec: failSpec, Handler: func(ctx context.Context, _ connect.Spec, stream connect.ServerStream) error {
			var req apiv1.WhoAmIRequest
			if err := stream.Receive(&req); err != nil {
				return err
			}
			info, _ := connect.CallInfoForServerContext(ctx)
			switch info.RequestHeader().Get("X-Failure") {
			case "domain":
				return apperr.New(apperr.NotFound, "TEST_NOT_FOUND", "test not found")
			case "local":
				return connect.NewError(connect.CodeInvalidArgument, "invalid input")
			case "panic":
				panic("secret panic")
			case "wait":
				<-ctx.Done()
				return ctx.Err()
			default:
				return errors.New("password=secret")
			}
		}})
		rpc.Register(connect.Method{Spec: tickSpec, Handler: func(ctx context.Context, _ connect.Spec, stream connect.ServerStream) error {
			var req apiv1.WhoAmIRequest
			if err := stream.Receive(&req); err != nil {
				return err
			}
			if err := stream.Send(&apiv1.WhoAmIResponse{Subject: "first"}); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(1100 * time.Millisecond):
			}
			return stream.Send(&apiv1.WhoAmIResponse{Subject: "second"})
		}})
		rpc.Register(connect.Method{Spec: sumSpec, Handler: func(_ context.Context, _ connect.Spec, stream connect.ServerStream) error {
			count := 0
			for {
				var req apiv1.WhoAmIRequest
				if err := stream.Receive(&req); err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					return err
				}
				count++
			}
			return stream.Send(&apiv1.WhoAmIResponse{Roles: make([]string, count)})
		}})
		plain.HandleFunc("GET /test/plain", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Subject", identity.FromContext(r.Context()).Subject)
		})
		plain.HandleFunc("GET /test/panic", func(http.ResponseWriter, *http.Request) { panic("secret") })
		plain.HandleFunc("POST /test/echo", func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "too large", 413)
				return
			}
			w.Write(body)
		})
	})
	require.NoError(t, err)
	return h
}

func testHTTPServer(t *testing.T, change ...func(*Deps)) *httptest.Server {
	t.Helper()
	srv := httptest.NewUnstartedServer(testHandler(t, change...))
	srv.Config.Protocols = New("", nil).Protocols
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

func clientContext(ctx context.Context, token string) (context.Context, *connect.CallInfo) {
	ctx, info := connect.NewClientContext(ctx)
	if token != "" {
		info.RequestHeader().Set("Authorization", "Bearer "+token)
	}
	return ctx, info
}

var _ middleware.Verifier = tokenVerifier{}
