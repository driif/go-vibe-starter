package reqlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/driif/go-vibe-starter/internal/reqlog"
)

func record(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	return rec
}

func TestHandler_AddsFields(t *testing.T) {
	const procedure = "/api.v1.WhoAmIService/WhoAmI"
	tests := []struct {
		name          string
		ctx           func() context.Context
		wantID        string
		wantProcedure string
		wantCode      string
	}{
		{
			name: "a context without fields adds nothing",
			ctx:  context.Background,
		},
		{
			name:   "the request id alone",
			ctx:    func() context.Context { return reqlog.With(context.Background(), "req-1") },
			wantID: "req-1",
		},
		{
			name: "the request id and the procedure, never the code",
			ctx: func() context.Context {
				ctx := reqlog.With(context.Background(), "req-2")
				reqlog.SetProcedure(ctx, procedure)
				reqlog.SetCode(ctx, "ok")
				return ctx
			},
			wantID: "req-2", wantProcedure: procedure, wantCode: "ok",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(reqlog.NewHandler(slog.NewJSONHandler(&buf, nil)))
			ctx := tt.ctx()

			logger.InfoContext(ctx, "hello", "k", "v")

			want := map[string]any{"level": "INFO", "msg": "hello", "k": "v"}
			if tt.wantID != "" {
				want["request_id"] = tt.wantID
			}
			if tt.wantProcedure != "" {
				want["procedure"] = tt.wantProcedure
			}
			rec := record(t, &buf)
			delete(rec, "time")
			assert.Equal(t, want, rec)
			assert.Equal(t, tt.wantProcedure, reqlog.Procedure(ctx))
			assert.Equal(t, tt.wantCode, reqlog.Code(ctx))
		})
	}
}

func TestSetters_WithoutFields(t *testing.T) {
	ctx := context.Background()

	reqlog.SetProcedure(ctx, "/api.v1.WhoAmIService/WhoAmI")
	reqlog.SetCode(ctx, "ok")

	assert.Empty(t, reqlog.Procedure(ctx))
	assert.Empty(t, reqlog.Code(ctx))
}

func TestHandler_WithAttrsKeepsWrapper(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(reqlog.NewHandler(slog.NewJSONHandler(&buf, nil))).
		With("component", "test").
		WithGroup("g")
	ctx := reqlog.With(context.Background(), "req-3")

	logger.InfoContext(ctx, "hello", "k", "v")

	rec := record(t, &buf)
	assert.Equal(t, "test", rec["component"])
	group, ok := rec["g"].(map[string]any)
	require.True(t, ok, "group g is missing: %v", rec)
	assert.Equal(t, "v", group["k"])
	assert.Equal(t, "req-3", group["request_id"])
}
