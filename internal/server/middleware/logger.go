package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// RequestBodyLogSkipper defines a function to skip logging certain request bodies.
type RequestBodyLogSkipper func(*http.Request) bool

// ResponseBodyLogSkipper defines a function to skip logging certain response bodies.
// The int argument is the response status code.
type ResponseBodyLogSkipper func(*http.Request, int) bool

// HeaderLogReplacer defines a function to sanitize headers before logging.
type HeaderLogReplacer func(http.Header) http.Header

// DefaultRequestBodyLogSkipper skips form and multipart bodies.
func DefaultRequestBodyLogSkipper(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	return strings.HasPrefix(ct, "application/x-www-form-urlencoded") ||
		strings.HasPrefix(ct, "multipart/form-data")
}

// DefaultResponseBodyLogSkipper skips everything except application/json.
func DefaultResponseBodyLogSkipper(_ *http.Request, _ int) bool {
	return false // caller checks Content-Type on the response writer
}

// DefaultHeaderLogReplacer redacts Authorization, X-CSRF-Token and Proxy-Authorization.
func DefaultHeaderLogReplacer(h http.Header) http.Header {
	out := http.Header{}
	for k, vv := range h {
		lk := strings.ToLower(k)
		redact := lk == "authorization" || lk == "x-csrf-token" || lk == "proxy-authorization"
		for _, v := range vv {
			if redact {
				out.Add(k, "*****REDACTED*****")
			} else {
				out.Add(k, v)
			}
		}
	}
	return out
}

// LoggerConfig configures the Logger middleware.
type LoggerConfig struct {
	Level                     slog.Level
	LogRequestBody            bool
	LogRequestHeader          bool
	LogRequestQuery           bool
	LogResponseBody           bool
	LogResponseHeader         bool
	Skipper                   func(*http.Request) bool
	RequestBodyLogSkipper     RequestBodyLogSkipper
	RequestHeaderLogReplacer  HeaderLogReplacer
	ResponseBodyLogSkipper    ResponseBodyLogSkipper
	ResponseHeaderLogReplacer HeaderLogReplacer
}

var DefaultLoggerConfig = LoggerConfig{
	Level:                     slog.LevelDebug,
	RequestBodyLogSkipper:     DefaultRequestBodyLogSkipper,
	RequestHeaderLogReplacer:  DefaultHeaderLogReplacer,
	ResponseBodyLogSkipper:    DefaultResponseBodyLogSkipper,
	ResponseHeaderLogReplacer: DefaultHeaderLogReplacer,
}

// Logger returns the middleware with DefaultLoggerConfig.
func Logger() func(http.Handler) http.Handler {
	return LoggerWithConfig(DefaultLoggerConfig)
}

// LoggerWithConfig returns a structured slog request/response logger.
func LoggerWithConfig(cfg LoggerConfig) func(http.Handler) http.Handler {
	if cfg.RequestBodyLogSkipper == nil {
		cfg.RequestBodyLogSkipper = DefaultRequestBodyLogSkipper
	}
	if cfg.RequestHeaderLogReplacer == nil {
		cfg.RequestHeaderLogReplacer = DefaultHeaderLogReplacer
	}
	if cfg.ResponseBodyLogSkipper == nil {
		cfg.ResponseBodyLogSkipper = DefaultResponseBodyLogSkipper
	}
	if cfg.ResponseHeaderLogReplacer == nil {
		cfg.ResponseHeaderLogReplacer = DefaultHeaderLogReplacer
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Skipper != nil && cfg.Skipper(r) {
				next.ServeHTTP(w, r)
				return
			}

			reqID := r.Header.Get("X-Request-Id")

			// --- request logging ---
			attrs := []slog.Attr{
				slog.String("id", reqID),
				slog.String("host", r.Host),
				slog.String("method", r.Method),
				slog.String("url", r.URL.String()),
				slog.String("bytes_in", r.Header.Get("Content-Length")),
			}

			if cfg.LogRequestBody && !cfg.RequestBodyLogSkipper(r) {
				if r.Body != nil {
					body, err := io.ReadAll(r.Body)
					if err == nil {
						r.Body = io.NopCloser(bytes.NewReader(body))
						attrs = append(attrs, slog.String("req_body", string(body)))
					}
				}
			}
			if cfg.LogRequestHeader {
				attrs = append(attrs, slog.Any("req_header", cfg.RequestHeaderLogReplacer(r.Header)))
			}
			if cfg.LogRequestQuery {
				attrs = append(attrs, slog.Any("req_query", r.URL.Query()))
			}

			slog.LogAttrs(r.Context(), cfg.Level, "request received", attrs...)

			// --- wrap response writer ---
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			var resBody bytes.Buffer
			if cfg.LogResponseBody {
				ww.Tee(&resBody)
			}

			start := time.Now()
			next.ServeHTTP(ww, r)
			duration := time.Since(start)

			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}

			// --- response logging ---
			resAttrs := []slog.Attr{
				slog.Int("status", status),
				slog.Int("bytes_out", ww.BytesWritten()),
				slog.Duration("duration_ms", duration),
			}

			if cfg.LogResponseBody && !cfg.ResponseBodyLogSkipper(r, status) {
				ct := ww.Header().Get("Content-Type")
				if strings.HasPrefix(ct, "application/json") {
					resAttrs = append(resAttrs, slog.String("res_body", resBody.String()))
				}
			}
			if cfg.LogResponseHeader {
				resAttrs = append(resAttrs, slog.Any("res_header", cfg.ResponseHeaderLogReplacer(ww.Header())))
			}

			slog.LogAttrs(r.Context(), cfg.Level, "response sent", resAttrs...)
		})
	}
}
