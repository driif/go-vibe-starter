package server

import (
	"connectrpc.com/connect/v2"
	"github.com/go-pkgz/routegroup"
)

func routes(rpc *connect.Server, plain *routegroup.Bundle, d Deps) {
	mountWhoAmI(rpc, plain, d)
	mountNotes(rpc, plain, d) // DEMO:line
}
