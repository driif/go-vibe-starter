// Package server composes the application HTTP and RPC endpoints.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/validate"
	"github.com/go-pkgz/routegroup"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/driif/go-vibe-starter/internal/rpc/interceptor"
	"github.com/driif/go-vibe-starter/internal/server/middleware"
)

const maxRequestBytes = 4 << 20

// Deps supplies the runtime dependencies without reading process configuration.
type Deps struct {
	DB          *pgxpool.Pool
	Verifier    middleware.Verifier
	CORSOrigins []string
	Metrics     prometheus.Registerer
}

// Handler constructs the application's public and authenticated routes.
func Handler(d Deps) (http.Handler, error) { return build(d, routes) }

func build(d Deps, hub func(*connect.Server, *routegroup.Bundle, Deps)) (http.Handler, error) {
	if d.DB == nil || d.Verifier == nil {
		return nil, errors.New("database and verifier are required")
	}
	router := routegroup.New(http.NewServeMux())
	access, err := middleware.AccessLog(d.Metrics, "/healthz", "/readyz")
	if err != nil {
		return nil, fmt.Errorf("register RPC metrics: %w", err)
	}
	router.Use(middleware.RequestID, access, middleware.Recover,
		middleware.SecurityHeaders, middleware.CORS(d.CORSOrigins))
	public := router.Group()
	public.HandleFunc("GET /healthz", healthz)
	public.HandleFunc("GET /readyz", readyz(d.DB, d.Verifier))
	authed := router.Group()
	authed.Use(middleware.Auth(d.Verifier))
	plain := authed.With(middleware.BodyLimit(maxRequestBytes))
	rpc := connect.NewServer(interceptor.RequestLog(), interceptor.Errors(), validate.NewServerInterceptor())
	hub(rpc, plain, d)
	connecthttp.Mount(authed, rpc, connecthttp.WithReadMaxBytes(maxRequestBytes))
	return router, nil
}

// New configures a server for Connect, gRPC and gRPC-Web on HTTP/1.1 and h2c.
func New(addr string, h http.Handler) *http.Server {
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	return &http.Server{
		Addr: addr, Handler: h, Protocols: protocols,
		ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 120 * time.Second,
		ErrorLog: slog.NewLogLogger(slog.Default().Handler(), slog.LevelWarn),
	}
}

// Serve runs all listeners and drains them together when canceled or one fails.
func Serve(ctx context.Context, grace time.Duration, servers map[*http.Server]net.Listener) error {
	if len(servers) == 0 {
		return errors.New("no servers to serve")
	}
	failures := make(chan error, len(servers))
	for srv, listener := range servers {
		go func() { failures <- srv.Serve(listener) }()
	}
	var serveErr error
	select {
	case <-ctx.Done():
	case err := <-failures:
		if !errors.Is(err, http.ErrServerClosed) {
			serveErr = err
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	var wg sync.WaitGroup
	shutdownErrors := make(chan error, len(servers))
	for srv := range servers {
		wg.Go(func() {
			if err := srv.Shutdown(shutdown); err != nil {
				shutdownErrors <- errors.Join(err, srv.Close())
			}
		})
	}
	wg.Wait()
	close(shutdownErrors)
	for err := range shutdownErrors {
		serveErr = errors.Join(serveErr, err)
	}
	return serveErr
}
