package test_test

import (
	"errors"
	"fmt"
	"testing"

	bufvalidate "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connectproto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/proto"

	"github.com/driif/go-vibe-starter/internal/test"
)

func TestReason(t *testing.T) {
	t.Parallel()
	detail, err := connectproto.NewErrorDetail(&errdetails.ErrorInfo{Reason: "TASK_NOT_FOUND", Domain: "api"})
	require.NoError(t, err)
	rpcErr := connect.NewError(connect.CodeNotFound, "task not found").WithDetail(detail)
	for _, tt := range []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"untyped", errors.New("failure"), ""},
		{"no details", connect.NewError(connect.CodeInternal, "internal"), ""},
		{"domain reason", rpcErr, "TASK_NOT_FOUND"},
		{"wrapped", fmt.Errorf("call: %w", rpcErr), "TASK_NOT_FOUND"},
		{"malformed detail", connect.NewError(connect.CodeUnknown, "unknown").WithDetail(&connect.ErrorDetail{Type: "google.rpc.ErrorInfo", Value: []byte{255}}), ""},
	} {
		t.Run(tt.name, func(t *testing.T) { assert.Equal(t, tt.want, test.Reason(tt.err)) })
	}
}

func TestViolations(t *testing.T) {
	t.Parallel()
	detail, err := connectproto.NewErrorDetail(&bufvalidate.Violations{Violations: []*bufvalidate.Violation{
		{Field: &bufvalidate.FieldPath{Elements: []*bufvalidate.FieldPathElement{{FieldName: proto.String("title")}}}},
		{Field: &bufvalidate.FieldPath{Elements: []*bufvalidate.FieldPathElement{{FieldName: proto.String("task")}, {FieldName: proto.String("id")}}}},
	}})
	require.NoError(t, err)
	rpcErr := connect.NewError(connect.CodeInvalidArgument, "invalid").WithDetail(detail)
	assert.Equal(t, []string{"title", "task.id"}, test.Violations(t, rpcErr))
	assert.Nil(t, test.Violations(t, nil))
	assert.Nil(t, test.Violations(t, errors.New("untyped")))
	assert.Empty(t, test.Violations(t, connect.NewError(connect.CodeInvalidArgument, "no details")))
}
