package middleware

import (
	"fmt"
	"net/http"
)

// SecureConfig holds settings for the security headers middleware.
type SecureConfig struct {
	XSSProtection         string
	ContentTypeNosniff    string
	XFrameOptions         string
	HSTSMaxAge            int
	HSTSExcludeSubdomains bool
	ContentSecurityPolicy string
	CSPReportOnly         bool
	HSTSPreloadEnabled    bool
	ReferrerPolicy        string
}

// Secure returns a middleware that sets security-related response headers.
func Secure(cfg SecureConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.XSSProtection != "" {
				w.Header().Set("X-XSS-Protection", cfg.XSSProtection)
			}
			if cfg.ContentTypeNosniff != "" {
				w.Header().Set("X-Content-Type-Options", cfg.ContentTypeNosniff)
			}
			if cfg.XFrameOptions != "" {
				w.Header().Set("X-Frame-Options", cfg.XFrameOptions)
			}
			if cfg.HSTSMaxAge > 0 {
				hstsVal := fmt.Sprintf("max-age=%d", cfg.HSTSMaxAge)
				if !cfg.HSTSExcludeSubdomains {
					hstsVal += "; includeSubDomains"
				}
				if cfg.HSTSPreloadEnabled {
					hstsVal += "; preload"
				}
				w.Header().Set("Strict-Transport-Security", hstsVal)
			}
			if cfg.ContentSecurityPolicy != "" {
				header := "Content-Security-Policy"
				if cfg.CSPReportOnly {
					header = "Content-Security-Policy-Report-Only"
				}
				w.Header().Set(header, cfg.ContentSecurityPolicy)
			}
			if cfg.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", cfg.ReferrerPolicy)
			}
			next.ServeHTTP(w, r)
		})
	}
}
