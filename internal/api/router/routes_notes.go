// DEMO: reference vertical slice — delete with `make init` or by hand.

package router

import (
	"github.com/driif/go-vibe-starter/internal/api/handlers"
	"github.com/driif/go-vibe-starter/internal/server"
	"github.com/go-chi/chi/v5"
)

// registerNotes mounts the notes slice behind the authenticator: every handler
// scopes its queries to the principal it finds in the request context.
func registerNotes(s *server.Server) {
	notes := handlers.NewNotes(s.DB)

	s.Router.Group(func(r chi.Router) {
		r.Use(s.Authenticator())
		r.Get("/v1/notes", notes.List)
		r.Post("/v1/notes", notes.Create)
		r.Get("/v1/notes/{id}", notes.Get)
		r.Delete("/v1/notes/{id}", notes.Delete)
	})
}
