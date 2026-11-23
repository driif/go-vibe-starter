package middleware_test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/driif/go-vibe-starter/internal/reqlog"
	"github.com/driif/go-vibe-starter/internal/server/middleware"
)

func TestRequestID(t *testing.T) {
	for _, id := range []string{"safe-id_123.abc", "", "unsafe id", strings.Repeat("a", 65)} {
		t.Run(id, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("X-Request-Id", id)
			w := httptest.NewRecorder()
			middleware.RequestID(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(w, r)
			actual := w.Header().Get("X-Request-Id")
			if id == "safe-id_123.abc" {
				require.Equal(t, id, actual)
			} else {
				require.Regexp(t, "^[a-f0-9]{32}$", actual)
			}
		})
	}
}

func TestAccessLog(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(reqlog.NewHandler(slog.NewJSONHandler(&output, nil))))
	t.Cleanup(func() { slog.SetDefault(previous) })
	reg := prometheus.NewRegistry()
	access := middleware.AccessLog(reg, "/healthz")
	h := middleware.RequestID(access(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rpc" {
			reqlog.SetCode(r.Context(), "ok")
			reqlog.SetProcedure(r.Context(), "/service/Method")
		}
		w.WriteHeader(http.StatusCreated)
	})))
	for _, path := range []string{"/healthz", "/plain", "/rpc?password=secret"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	require.Len(t, lines, 2)
	var record map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &record))
	require.Equal(t, "request", record["msg"])
	require.Equal(t, "/rpc", record["path"])
	require.EqualValues(t, 201, record["status"])
	require.NotEmpty(t, record["request_id"])
	require.Contains(t, record, "duration_ms")
	require.NotContains(t, output.String(), "password")
	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	require.Equal(t, "rpc_server_duration_seconds", families[0].GetName())
	require.Len(t, families[0].Metric, 1)
	require.EqualValues(t, 1, families[0].Metric[0].GetHistogram().GetSampleCount())
	labels := families[0].Metric[0].Label
	require.Equal(t, "ok", labels[0].GetValue())
	require.Equal(t, "/service/Method", labels[1].GetValue())
}

func TestAccessLog_WriterKeepsFlush(t *testing.T) {
	w := httptest.NewRecorder()
	middleware.AccessLog(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)
		flusher.Flush()
	})).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	require.True(t, w.Flushed)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestRecover(t *testing.T) {
	w := httptest.NewRecorder()
	middleware.Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("secret") })).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, 500, w.Code)
	require.Equal(t, "Internal Server Error\n", w.Body.String())
	require.PanicsWithValue(t, http.ErrAbortHandler, func() {
		middleware.Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) })).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	})
}

func TestSecurityHeaders(t *testing.T) {
	for _, mode := range []string{"http", "tls", "proxy"} {
		t.Run(mode, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if mode == "tls" {
				r.TLS = &tls.ConnectionState{}
			}
			if mode == "proxy" {
				r.Header.Set("X-Forwarded-Proto", "https")
			}
			w := httptest.NewRecorder()
			middleware.SecurityHeaders(http.NotFoundHandler()).ServeHTTP(w, r)
			require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
			require.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
			require.Equal(t, "no-referrer", w.Header().Get("Referrer-Policy"))
			require.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'none'")
			require.Equal(t, mode != "http", w.Header().Get("Strict-Transport-Security") != "")
		})
	}
}

func TestCORS(t *testing.T) {
	for _, tt := range []struct {
		name, origin, method string
		status               int
	}{
		{"allowed preflight", "https://app.example", http.MethodOptions, 204},
		{"disallowed preflight", "https://other.example", http.MethodOptions, 401},
		{"allowed error", "https://app.example", http.MethodPost, 401},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "/", nil)
			r.Header.Set("Origin", tt.origin)
			r.Header.Set("Access-Control-Request-Method", "POST")
			w := httptest.NewRecorder()
			middleware.CORS([]string{"https://app.example"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(401) })).ServeHTTP(w, r)
			require.Equal(t, tt.status, w.Code)
			if tt.origin == "https://app.example" {
				require.Equal(t, tt.origin, w.Header().Get("Access-Control-Allow-Origin"))
				require.Contains(t, w.Header().Get("Access-Control-Expose-Headers"), "X-Request-Id")
			} else {
				require.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
			}
			require.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
			if tt.status == 204 {
				require.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
				require.Equal(t, "7200", w.Header().Get("Access-Control-Max-Age"))
			}
		})
	}
}

func TestBodyLimit(t *testing.T) {
	h := middleware.BodyLimit(4)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	for _, tt := range []struct {
		body   string
		status int
	}{{"1234", 200}, {"12345", 413}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body)))
		require.Equal(t, tt.status, w.Code)
	}
}
