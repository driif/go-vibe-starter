package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/driif/go-vibe-starter/internal/server"
	"github.com/driif/go-vibe-starter/internal/server/config"
	"github.com/go-pkgz/rest"
)

// readinessTimeout bounds every readiness check so a hung dependency cannot hold
// the probe open until the client gives up.
const readinessTimeout = 2 * time.Second

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type readyResponse struct {
	Status string `json:"status"`
	// Check names the dependency that failed; empty when everything is up.
	Check string `json:"check,omitempty"`
	Error string `json:"error,omitempty"`
}

// Health reports that the process is alive. It never touches a dependency, so a
// failing database must not take the container down.
func Health(w http.ResponseWriter, _ *http.Request) {
	rest.RenderJSON(w, healthResponse{Status: "ok", Version: config.GetFormattedBuildArgs()})
}

// Ready reports whether the process can serve traffic: 200 once the database
// answers within readinessTimeout, 503 naming the failing check otherwise.
func Ready(s *server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.DB == nil {
			renderUnavailable(w, "database", "not initialized")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()

		if err := s.DB.PingContext(ctx); err != nil {
			renderUnavailable(w, "database", err.Error())
			return
		}

		rest.RenderJSON(w, readyResponse{Status: "ok"})
	}
}

// renderUnavailable writes a 503. The content type is set before WriteHeader
// because rest.RenderJSON sets it too late once the status is already out.
func renderUnavailable(w http.ResponseWriter, check, reason string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	rest.RenderJSON(w, readyResponse{Status: "unavailable", Check: check, Error: reason})
}
