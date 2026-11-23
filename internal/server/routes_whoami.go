package server

import (
	"connectrpc.com/connect/v2"
	"github.com/go-pkgz/routegroup"

	"github.com/driif/go-vibe-starter/internal/gen/api/v1/apiv1connect"
	"github.com/driif/go-vibe-starter/internal/rpc/whoami"
)

func mountWhoAmI(rpc *connect.Server, _ *routegroup.Bundle, _ Deps) {
	apiv1connect.RegisterWhoAmIServiceHandler(rpc, whoami.Handler{})
}
