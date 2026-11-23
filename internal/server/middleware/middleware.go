// Package middleware provides the HTTP concerns shared by RPCs and plain routes.
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"connectrpc.com/cors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/driif/go-vibe-starter/internal/reqlog"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// RequestID attaches a bounded request identifier to the response and log context.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if !requestIDPattern.MatchString(id) {
			var value [16]byte
			rand.Read(value[:])
			id = hex.EncodeToString(value[:])
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(reqlog.With(r.Context(), id)))
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	if status >= 100 && status < 200 {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}

func (w *responseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *responseWriter) Flush() {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if err := http.NewResponseController(w.ResponseWriter).Flush(); err != nil {
		slog.Debug("flush response", "error", err)
	}
}

// AccessLog records one access line and, for RPC calls, a duration observation.
func AccessLog(reg prometheus.Registerer, skip ...string) (func(http.Handler) http.Handler, error) {
	var duration *prometheus.HistogramVec
	if reg != nil {
		duration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "rpc_server_duration_seconds",
			Help:    "RPC duration including authentication and HTTP middleware.",
			Buckets: prometheus.DefBuckets,
		}, []string{"procedure", "code"})
		if err := reg.Register(duration); err != nil {
			return nil, err
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if slices.Contains(skip, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			start := time.Now()
			wrapped := &responseWriter{ResponseWriter: w}
			next.ServeHTTP(wrapped, r)
			elapsed := time.Since(start)
			status := wrapped.status
			if status == 0 {
				status = http.StatusOK
			}
			attrs := []slog.Attr{
				slog.String("method", r.Method), slog.String("path", r.URL.Path),
				slog.Int("status", status), slog.Int64("duration_ms", elapsed.Milliseconds()),
			}
			if code := reqlog.Code(r.Context()); code != "" {
				attrs = append(attrs, slog.String("code", code))
				if duration != nil {
					duration.WithLabelValues(reqlog.Procedure(r.Context()), code).Observe(elapsed.Seconds())
				}
			}
			slog.LogAttrs(r.Context(), slog.LevelInfo, "request", attrs...)
		})
	}, nil
}

// Recover converts a plain HTTP handler panic into an internal error response.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if value := recover(); value != nil {
				if err, ok := value.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(value)
				}
				slog.ErrorContext(r.Context(), "panic", "panic", value, "stack", string(debug.Stack()))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders applies conservative API response headers.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

// CORS permits only explicitly listed browser origins.
func CORS(origins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if len(origins) != 0 {
				w.Header().Add("Vary", "Origin")
			}
			if origin != "" && slices.Contains(origins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(append(cors.ExposedHeaders(), "X-Request-Id"), ", "))
				if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
					w.Header().Add("Vary", "Access-Control-Request-Method")
					w.Header().Add("Vary", "Access-Control-Request-Headers")
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(cors.AllowedMethods(), ", "))
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(append(cors.AllowedHeaders(), "Authorization"), ", "))
					w.Header().Set("Access-Control-Max-Age", "7200")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BodyLimit bounds a plain HTTP request body without limiting an entire RPC stream.
func BodyLimit(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return http.MaxBytesHandler(next, n) }
}
