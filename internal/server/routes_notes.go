// DEMO: notes
package server

import (
	"connectrpc.com/connect/v2"
	"github.com/go-pkgz/routegroup"

	"github.com/driif/go-vibe-starter/internal/domain/notes"
	"github.com/driif/go-vibe-starter/internal/gen/api/v1/apiv1connect"
	rpcnotes "github.com/driif/go-vibe-starter/internal/rpc/notes"
)

func mountNotes(rpc *connect.Server, _ *routegroup.Bundle, d Deps) {
	apiv1connect.RegisterNotesServiceHandler(rpc, rpcnotes.New(notes.NewStore(d.DB)))
}
