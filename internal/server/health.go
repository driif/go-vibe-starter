package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type pinger interface{ Ping(context.Context) error }
type readier interface{ Ready(context.Context) error }

func healthz(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

func readyz(db pinger, verifier any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			slog.WarnContext(ctx, "database not ready", "error", err)
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		if v, ok := verifier.(readier); ok {
			if err := v.Ready(ctx); err != nil {
				slog.WarnContext(ctx, "identity provider not ready", "error", err)
				http.Error(w, "identity provider unavailable", http.StatusServiceUnavailable)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}
