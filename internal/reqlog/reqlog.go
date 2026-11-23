// Package reqlog carries the log fields of one request: the request id from the HTTP middleware and
// the procedure and Connect code from the RPC interceptor.
package reqlog

import (
	"context"
	"log/slog"
	"sync"
)

type key struct{}

// fields is written by the middleware and the interceptor and read by every log call of the
// request, possibly from goroutines the handler started.
type fields struct {
	mu        sync.Mutex
	id        string
	procedure string
	code      string
}

// With returns ctx carrying new fields for the request with the given id.
func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, key{}, &fields{id: id})
}

func from(ctx context.Context) *fields {
	f, _ := ctx.Value(key{}).(*fields)
	return f
}

// SetProcedure records the RPC procedure. Without fields in ctx it does nothing.
func SetProcedure(ctx context.Context, procedure string) {
	f := from(ctx)
	if f == nil {
		return
	}
	f.mu.Lock()
	f.procedure = procedure
	f.mu.Unlock()
}

// SetCode records the Connect code the client gets: "ok" or the name of the error code. Without
// fields in ctx it does nothing.
func SetCode(ctx context.Context, code string) {
	f := from(ctx)
	if f == nil {
		return
	}
	f.mu.Lock()
	f.code = code
	f.mu.Unlock()
}

// Procedure returns the procedure SetProcedure recorded, or "" when unset.
func Procedure(ctx context.Context) string {
	f := from(ctx)
	if f == nil {
		return ""
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.procedure
}

// Code returns the code SetCode recorded, or "" when unset.
func Code(ctx context.Context) string {
	f := from(ctx)
	if f == nil {
		return ""
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.code
}

// NewHandler returns a handler that adds request_id and, when set, procedure from the context to
// each record.
func NewHandler(h slog.Handler) slog.Handler {
	return handler{h}
}

type handler struct {
	slog.Handler
}

func (h handler) Handle(ctx context.Context, r slog.Record) error {
	f := from(ctx)
	if f == nil {
		return h.Handler.Handle(ctx, r)
	}
	r = r.Clone()
	f.mu.Lock()
	r.AddAttrs(slog.String("request_id", f.id))
	if f.procedure != "" {
		r.AddAttrs(slog.String("procedure", f.procedure))
	}
	f.mu.Unlock()
	return h.Handler.Handle(ctx, r)
}

func (h handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return handler{h.Handler.WithAttrs(attrs)}
}

func (h handler) WithGroup(name string) slog.Handler {
	return handler{h.Handler.WithGroup(name)}
}
