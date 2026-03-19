package middleware

// Based on https://github.com/LYY/echo-middleware (MIT License)
// Ported from Goji's middleware: https://github.com/zenazn/goji/tree/master/web/middleware

import (
	"net/http"
	"time"
)

var (
	epoch = time.Unix(0, 0).Format(time.RFC1123)

	noCacheHeaders = map[string]string{
		"Expires":         epoch,
		"Cache-Control":   "no-cache, private, max-age=0",
		"Pragma":          "no-cache",
		"X-Accel-Expires": "0",
	}
	etagHeaders = []string{
		"ETag",
		"If-Modified-Since",
		"If-Match",
		"If-None-Match",
		"If-Range",
		"If-Unmodified-Since",
	}
)

// NoCache sets headers to prevent caching by proxies and clients.
func NoCache(next http.Handler) http.Handler {
	return NoCacheWithSkipper(nil)(next)
}

// NoCacheWithSkipper returns a NoCache middleware with an optional skipper.
// If skip returns true for a request, the middleware is bypassed.
func NoCacheWithSkipper(skip func(*http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skip != nil && skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			for _, h := range etagHeaders {
				r.Header.Del(h)
			}
			for k, v := range noCacheHeaders {
				w.Header().Set(k, v)
			}

			next.ServeHTTP(w, r)
		})
	}
}
