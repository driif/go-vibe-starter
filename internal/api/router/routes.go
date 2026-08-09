package router

import "github.com/driif/go-vibe-starter/internal/server"

// RegisterHandlersV1 mounts every domain on the server router. Each domain owns
// one routes_*.go file, so adding a domain never edits a shared file.
func RegisterHandlersV1(s *server.Server) {
	registerHealth(s)
	registerUsers(s)
	registerNotes(s)
}
