package test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	bufvalidate "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/connect/v2/connectproto"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/driif/go-vibe-starter/internal/db"
	"github.com/driif/go-vibe-starter/internal/identity/keycloak"
	"github.com/driif/go-vibe-starter/internal/identity/testprovider"
	"github.com/driif/go-vibe-starter/internal/server"
)

// Server is a production handler with isolated database and identity fixtures.
type Server struct {
	URL      string
	DB       *pgxpool.Pool
	Provider *testprovider.Provider
	Metrics  *prometheus.Registry
}

// NewDeps builds the dependencies used by the production HTTP stack.
func NewDeps(t *testing.T) (server.Deps, *Server) {
	t.Helper()
	pool, err := db.Open(t.Context(), Database(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	provider := testprovider.New()
	t.Cleanup(provider.Close)
	fixture := &Server{DB: pool, Provider: provider, Metrics: prometheus.NewRegistry()}
	deps := server.Deps{DB: pool, Verifier: keycloak.New(provider.URL(), testprovider.Audience), Metrics: fixture.Metrics}
	return deps, fixture
}

// NewHandler constructs the production handler, applying dependency changes first.
func NewHandler(t *testing.T, change ...func(*server.Deps)) (http.Handler, *Server) {
	t.Helper()
	deps, fixture := NewDeps(t)
	for _, fn := range change {
		fn(&deps)
	}
	handler, err := server.Handler(deps)
	require.NoError(t, err)
	return handler, fixture
}

// NewServer serves the production handler over HTTP/1.1 and h2c.
func NewServer(t *testing.T, change ...func(*server.Deps)) *Server {
	t.Helper()
	handler, fixture := NewHandler(t, change...)
	srv := httptest.NewUnstartedServer(handler)
	srv.Config.Protocols = server.New("", nil).Protocols
	srv.Start()
	t.Cleanup(srv.Close)
	fixture.URL = srv.URL
	return fixture
}

// Token returns a signed access token for subject and its realm roles.
func (s *Server) Token(subject string, roles ...string) string {
	return s.Provider.Token(s.Provider.Claims(subject, roles...))
}

// Client returns a Connect client using token and the selected wire options.
func (s *Server) Client(token string, opts ...connecthttp.Option) *connect.Client {
	return connect.NewClient(connecthttp.NewTransport(HTTPClient(token), s.URL, opts...))
}

// HTTPClient supports HTTP/1.1 and h2c and adds a bearer header when token is nonempty.
func HTTPClient(token string) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Protocols = new(http.Protocols)
	transport.Protocols.SetHTTP1(true)
	transport.Protocols.SetHTTP2(true)
	transport.Protocols.SetUnencryptedHTTP2(true)
	return &http.Client{Transport: bearerTransport{base: transport, token: token}}
}

type bearerTransport struct {
	base  *http.Transport
	token string
}

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if b.token != "" {
		r = r.Clone(r.Context())
		r.Header.Set("Authorization", "Bearer "+b.token)
	}
	return b.base.RoundTrip(r)
}

// Reason returns the first ErrorInfo reason attached to a Connect error, or empty.
func Reason(err error) string {
	var rpcErr *connect.Error
	if !errors.As(err, &rpcErr) {
		return ""
	}
	for _, detail := range rpcErr.Details() {
		message, decodeErr := connectproto.UnmarshalErrorDetail(detail)
		if decodeErr == nil {
			if info, ok := message.(*errdetails.ErrorInfo); ok {
				return info.GetReason()
			}
		}
	}
	return ""
}

// Violations returns field paths from Protovalidate details attached to err.
func Violations(t *testing.T, err error) []string {
	t.Helper()
	var rpcErr *connect.Error
	if !errors.As(err, &rpcErr) {
		return nil
	}
	var fields []string
	for _, detail := range rpcErr.Details() {
		message, decodeErr := connectproto.UnmarshalErrorDetail(detail)
		require.NoError(t, decodeErr)
		violations, ok := message.(*bufvalidate.Violations)
		if !ok {
			continue
		}
		for _, violation := range violations.GetViolations() {
			var path []string
			for _, element := range violation.GetField().GetElements() {
				path = append(path, element.GetFieldName())
			}
			fields = append(fields, strings.Join(path, "."))
		}
	}
	return fields
}
