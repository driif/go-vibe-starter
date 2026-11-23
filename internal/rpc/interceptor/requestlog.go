// Package interceptor provides shared RPC error and request-log handling.
package interceptor

import (
	"context"

	"connectrpc.com/connect/v2"
	"github.com/driif/go-vibe-starter/internal/reqlog"
)

// RequestLog records the registered procedure and final RPC code in the request context.
func RequestLog() connect.ServerInterceptor {
	return func(next connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) error {
			reqlog.SetProcedure(ctx, spec.Procedure)
			err := next(ctx, spec, stream)
			code := "ok"
			if err != nil {
				code = connect.CodeOf(err).String()
			}
			reqlog.SetCode(ctx, code)
			return err
		}
	}
}
