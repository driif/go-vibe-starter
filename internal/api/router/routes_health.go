package router

import (
	"github.com/driif/go-vibe-starter/internal/api/handlers"
	"github.com/driif/go-vibe-starter/internal/server"
)

// registerHealth mounts the probes. They stay outside the auth group: an
// orchestrator has no token to present.
func registerHealth(s *server.Server) {
	s.Router.Get("/health", handlers.Health)
	s.Router.Get("/ready", handlers.Ready(s))
}
