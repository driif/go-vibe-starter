package interceptor_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connectproto"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/driif/go-vibe-starter/internal/apperr"
	"github.com/driif/go-vibe-starter/internal/reqlog"
	"github.com/driif/go-vibe-starter/internal/rpc/interceptor"
)

func TestErrors_Mapping(t *testing.T) {
	local := connect.NewError(connect.CodeInvalidArgument, "local validation")
	tests := []struct {
		name    string
		err     error
		code    connect.Code
		message string
	}{
		{"internal", apperr.New(apperr.Internal, "INTERNAL", "domain message"), connect.CodeInternal, "domain message"},
		{"invalid argument", apperr.New(apperr.InvalidArgument, "INVALID", "domain message"), connect.CodeInvalidArgument, "domain message"},
		{"not found", apperr.New(apperr.NotFound, "NOT_FOUND", "domain message"), connect.CodeNotFound, "domain message"},
		{"already exists", apperr.New(apperr.AlreadyExists, "EXISTS", "domain message"), connect.CodeAlreadyExists, "domain message"},
		{"permission", apperr.New(apperr.PermissionDenied, "DENIED", "domain message"), connect.CodePermissionDenied, "domain message"},
		{"precondition", apperr.New(apperr.FailedPrecondition, "PRECONDITION", "domain message"), connect.CodeFailedPrecondition, "domain message"},
		{"unknown kind", apperr.New(apperr.Kind(100), "UNKNOWN", "domain message"), connect.CodeInternal, "domain message"},
		{"wrapped domain", fmt.Errorf("database: %w", apperr.New(apperr.NotFound, "NOT_FOUND", "domain message")), connect.CodeNotFound, "domain message"},
		{"local connect", local, connect.CodeInvalidArgument, "local validation"},
		{"remote connect", local.WithRemote(), connect.CodeInternal, "internal error"},
		{"canceled", context.Canceled, connect.CodeCanceled, "canceled"},
		{"deadline", context.DeadlineExceeded, connect.CodeDeadlineExceeded, "deadline exceeded"},
		{"raw", errors.New("password=secret"), connect.CodeInternal, "internal error"},
	}
	for _, streamType := range []connect.StreamType{connect.StreamTypeUnary, connect.StreamTypeServer, connect.StreamTypeClient, connect.StreamTypeBidi} {
		for _, tt := range tests {
			t.Run(streamType.String()+"/"+tt.name, func(t *testing.T) {
				fn := interceptor.Errors()(func(context.Context, connect.Spec, connect.ServerStream) error { return tt.err })
				err := fn(t.Context(), connect.Spec{StreamType: streamType}, nil)
				require.Equal(t, tt.code, connect.CodeOf(err))
				var rpc *connect.Error
				require.ErrorAs(t, err, &rpc)
				require.Equal(t, tt.message, rpc.Message())
				var domain *apperr.Error
				if errors.As(tt.err, &domain) {
					require.Len(t, rpc.Details(), 1)
					detail, err := connectproto.UnmarshalErrorDetail(rpc.Details()[0])
					require.NoError(t, err)
					info := detail.(*errdetails.ErrorInfo)
					require.Equal(t, domain.Reason, info.GetReason())
					require.Equal(t, "api", info.GetDomain())
				}
			})
		}
	}
	t.Run("panic", func(t *testing.T) {
		fn := interceptor.Errors()(func(context.Context, connect.Spec, connect.ServerStream) error { panic("secret") })
		err := fn(t.Context(), connect.Spec{}, nil)
		require.Equal(t, "internal: internal error", err.Error())
	})
	t.Run("success", func(t *testing.T) {
		fn := interceptor.Errors()(func(context.Context, connect.Spec, connect.ServerStream) error { return nil })
		require.NoError(t, fn(t.Context(), connect.Spec{}, nil))
	})
}

func TestRequestLog(t *testing.T) {
	for _, tt := range []struct {
		err  error
		code string
	}{
		{nil, "ok"}, {connect.NewError(connect.CodeNotFound, "missing"), "not_found"},
	} {
		t.Run(tt.code, func(t *testing.T) {
			ctx := reqlog.With(t.Context(), "id")
			fn := interceptor.RequestLog()(func(ctx context.Context, _ connect.Spec, _ connect.ServerStream) error {
				require.Equal(t, "/service/Method", reqlog.Procedure(ctx))
				return tt.err
			})
			require.Equal(t, tt.err, fn(ctx, connect.Spec{Procedure: "/service/Method"}, nil))
			require.Equal(t, tt.code, reqlog.Code(ctx))
		})
	}
}
