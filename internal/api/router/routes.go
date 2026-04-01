package router

import (
	"github.com/driif/go-vibe-starter/internal/api/handlers"
	"github.com/driif/go-vibe-starter/internal/server"
	"github.com/driif/go-vibe-starter/internal/server/auth"
	"github.com/go-chi/chi/v5"
)

func RegisterHandlersV1(s *server.Server) {
	s.Router.Group(func(r chi.Router) {
		r.Use(auth.Authenticate(s.Auth, auth.Options{}))
		r.Get("/v1/users/me", handlers.GetMe)
		r.With(auth.RequireRealmRoles(false, "admin")).Get("/v1/users", handlers.ListUsers(s))
	})
}
