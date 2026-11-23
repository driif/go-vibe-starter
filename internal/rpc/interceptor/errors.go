package interceptor

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connectproto"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/driif/go-vibe-starter/internal/apperr"
)

var kindCodes = map[apperr.Kind]connect.Code{
	apperr.Internal:           connect.CodeInternal,
	apperr.InvalidArgument:    connect.CodeInvalidArgument,
	apperr.NotFound:           connect.CodeNotFound,
	apperr.AlreadyExists:      connect.CodeAlreadyExists,
	apperr.PermissionDenied:   connect.CodePermissionDenied,
	apperr.FailedPrecondition: connect.CodeFailedPrecondition,
}

// Errors maps domain failures to RPC status and contains handler panics.
func Errors() connect.ServerInterceptor {
	return func(next connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) (result error) {
			defer func() {
				if value := recover(); value != nil {
					slog.ErrorContext(ctx, "rpc panic", "panic", value, "stack", string(debug.Stack()))
					result = connect.NewError(connect.CodeInternal, "internal error")
				}
			}()
			err := next(ctx, spec, stream)
			if err == nil {
				return nil
			}
			var domain *apperr.Error
			if errors.As(err, &domain) {
				code, ok := kindCodes[domain.Kind]
				if !ok {
					code = connect.CodeInternal
				}
				if code == connect.CodeInternal {
					slog.ErrorContext(ctx, "rpc failed", "error", err)
				}
				mapped := connect.NewError(code, domain.Message)
				detail, detailErr := connectproto.NewErrorDetail(&errdetails.ErrorInfo{Reason: domain.Reason, Domain: "api"})
				if detailErr == nil {
					mapped = mapped.WithDetail(detail)
				}
				return mapped
			}
			var rpc *connect.Error
			if errors.As(err, &rpc) {
				if !rpc.IsRemote() {
					return rpc
				}
			} else if errors.Is(err, context.Canceled) {
				slog.DebugContext(ctx, "rpc canceled", "error", err)
				return connect.NewError(connect.CodeCanceled, "canceled")
			} else if errors.Is(err, context.DeadlineExceeded) {
				slog.DebugContext(ctx, "rpc deadline exceeded", "error", err)
				return connect.NewError(connect.CodeDeadlineExceeded, "deadline exceeded")
			}
			slog.ErrorContext(ctx, "rpc failed", "error", err)
			return connect.NewError(connect.CodeInternal, "internal error")
		}
	}
}
