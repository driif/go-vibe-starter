// Package whoami serves api.v1.WhoAmIService.
package whoami

import (
	"context"

	"connectrpc.com/connect/v2"

	apiv1 "github.com/driif/go-vibe-starter/internal/gen/api/v1"
	"github.com/driif/go-vibe-starter/internal/identity"
)

type Handler struct{}

func (Handler) WhoAmI(ctx context.Context, _ *apiv1.WhoAmIRequest) (*apiv1.WhoAmIResponse, error) {
	p := identity.FromContext(ctx)
	if p == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, "no principal")
	}
	return &apiv1.WhoAmIResponse{
		Subject:  p.Subject,
		Username: p.Username,
		Email:    p.Email,
		Name:     p.Name,
		Roles:    p.Roles,
	}, nil
}
