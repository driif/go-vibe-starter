// DEMO: notes

// Package notes serves api.v1.NotesService.
package notes

import (
	"context"

	"connectrpc.com/connect/v2"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/driif/go-vibe-starter/internal/apperr"
	"github.com/driif/go-vibe-starter/internal/domain/notes"
	apiv1 "github.com/driif/go-vibe-starter/internal/gen/api/v1"
	"github.com/driif/go-vibe-starter/internal/identity"
)

// Handler implements apiv1connect.NotesServiceHandler over a notes store.
type Handler struct {
	store *notes.Store
}

// New returns a Handler over store.
func New(store *notes.Store) *Handler { return &Handler{store: store} }

// CreateNote stores a note of the caller.
func (h *Handler) CreateNote(ctx context.Context, req *apiv1.CreateNoteRequest) (*apiv1.CreateNoteResponse, error) {
	owner, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	note, err := h.store.Create(ctx, owner, req.GetTitle(), req.GetBody())
	if err != nil {
		return nil, err
	}
	return &apiv1.CreateNoteResponse{Note: toProto(note)}, nil
}

// GetNote returns a note of the caller.
func (h *Handler) GetNote(ctx context.Context, req *apiv1.GetNoteRequest) (*apiv1.GetNoteResponse, error) {
	owner, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}
	note, err := h.store.Get(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	return &apiv1.GetNoteResponse{Note: toProto(note)}, nil
}

// ListNotes returns the caller's newest notes.
func (h *Handler) ListNotes(ctx context.Context, _ *apiv1.ListNotesRequest) (*apiv1.ListNotesResponse, error) {
	owner, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	list, err := h.store.List(ctx, owner)
	if err != nil {
		return nil, err
	}
	resp := &apiv1.ListNotesResponse{Notes: make([]*apiv1.Note, 0, len(list))}
	for _, note := range list {
		resp.Notes = append(resp.Notes, toProto(note))
	}
	return resp, nil
}

// DeleteNote removes a note of the caller.
func (h *Handler) DeleteNote(ctx context.Context, req *apiv1.DeleteNoteRequest) (*apiv1.DeleteNoteResponse, error) {
	owner, err := subject(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID(req.GetId())
	if err != nil {
		return nil, err
	}
	if err := h.store.Delete(ctx, owner, id); err != nil {
		return nil, err
	}
	return &apiv1.DeleteNoteResponse{}, nil
}

func subject(ctx context.Context) (string, error) {
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

func toProto(n notes.Note) *apiv1.Note {
	return &apiv1.Note{Id: n.ID.String(), Title: n.Title, Body: n.Body, CreateTime: timestamppb.New(n.CreatedAt)}
}
