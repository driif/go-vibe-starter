// DEMO: notes

// Package notes serves the NotesService.
package notes

import (
	"context"

	"connectrpc.com/connect/v2"
	"github.com/driif/go-vibe-starter/internal/apperr"
	"github.com/driif/go-vibe-starter/internal/domain/notes"
	apiv1 "github.com/driif/go-vibe-starter/internal/gen/api/v1"
	"github.com/driif/go-vibe-starter/internal/identity"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct{ store *notes.Store }

func New(store *notes.Store) *Handler { return &Handler{store} }
func toProto(n notes.Note) *apiv1.Note {
	return &apiv1.Note{Id: n.ID.String(), Title: n.Title, Body: n.Body, CreateTime: timestamppb.New(n.CreatedAt)}
}
func owner(ctx context.Context) (string, error) {
	p := identity.FromContext(ctx)
	if p == nil {
		return "", connect.NewError(connect.CodeUnauthenticated, "no principal")
	}
	return p.Subject, nil
}
func parseID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, apperr.New(apperr.InvalidArgument, "NOTE_ID_INVALID", "id is not a UUID")
	}
	return id, nil
}
func (h *Handler) CreateNote(ctx context.Context, req *apiv1.CreateNoteRequest) (*apiv1.CreateNoteResponse, error) {
	subject, err := owner(ctx)
	if err != nil {
		return nil, err
	}
	n, err := h.store.Create(ctx, subject, req.GetTitle(), req.GetBody())
	if err != nil {
		return nil, err
	}
	return &apiv1.CreateNoteResponse{Note: toProto(n)}, nil
}
func (h *Handler) GetNote(ctx context.Context, req *apiv1.GetNoteRequest) (*apiv1.GetNoteResponse, error) {
	subject, err := owner(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}
	n, err := h.store.Get(ctx, subject, id)
	if err != nil {
		return nil, err
	}
	return &apiv1.GetNoteResponse{Note: toProto(n)}, nil
}
func (h *Handler) ListNotes(ctx context.Context, _ *apiv1.ListNotesRequest) (*apiv1.ListNotesResponse, error) {
	subject, err := owner(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.store.List(ctx, subject)
	if err != nil {
		return nil, err
	}
	resp := &apiv1.ListNotesResponse{Notes: make([]*apiv1.Note, 0, len(rows))}
	for _, row := range rows {
		resp.Notes = append(resp.Notes, toProto(row))
	}
	return resp, nil
}
func (h *Handler) DeleteNote(ctx context.Context, req *apiv1.DeleteNoteRequest) (*apiv1.DeleteNoteResponse, error) {
	subject, err := owner(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}
	if err := h.store.Delete(ctx, subject, id); err != nil {
		return nil, err
	}
	return &apiv1.DeleteNoteResponse{}, nil
}
