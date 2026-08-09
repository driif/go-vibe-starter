// DEMO: reference vertical slice — delete with `make init` or by hand.

package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/driif/go-vibe-starter/internal/api"
	"github.com/driif/go-vibe-starter/internal/db/gen"
	"github.com/driif/go-vibe-starter/internal/server/auth"
	"github.com/driif/go-vibe-starter/internal/server/errs"
	"github.com/go-chi/chi/v5"
	"github.com/go-pkgz/rest"
	"github.com/google/uuid"
)

// maxNoteTitleLen mirrors the maxLength in oapi/openapi.yaml.
const maxNoteTitleLen = 200

// noteStore is the slice of the sqlc-generated querier these handlers use.
// Declared at the consumer, so a test can substitute a fake without a database.
type noteStore interface {
	CreateNote(ctx context.Context, arg gen.CreateNoteParams) (gen.Note, error)
	GetNote(ctx context.Context, arg gen.GetNoteParams) (gen.Note, error)
	ListNotesByOwner(ctx context.Context, ownerID string) ([]gen.Note, error)
	DeleteNote(ctx context.Context, arg gen.DeleteNoteParams) error
}

// Notes serves the notes endpoints. Every query carries the caller's principal
// subject as owner_id, so one caller can never reach another's rows.
type Notes struct {
	store noteStore
}

// NewNotes builds the notes handlers over the generated queries.
func NewNotes(db gen.DBTX) *Notes {
	return &Notes{store: gen.New(db)}
}

// List returns the caller's notes, newest first.
func (h *Notes) List(w http.ResponseWriter, r *http.Request) {
	owner, ok := noteOwner(w, r)
	if !ok {
		return
	}

	notes, err := h.store.ListNotesByOwner(r.Context(), owner)
	if err != nil {
		writeNoteStoreError(w, r, err)
		return
	}

	out := make([]api.Note, len(notes))
	for i, n := range notes {
		out[i] = toAPINote(n)
	}
	rest.RenderJSON(w, out)
}

// Create stores a note owned by the caller.
func (h *Notes) Create(w http.ResponseWriter, r *http.Request) {
	owner, ok := noteOwner(w, r)
	if !ok {
		return
	}

	var req api.CreateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errs.Write(w, http.StatusBadRequest, fmt.Errorf("decode body: %w", err))
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" || len(title) > maxNoteTitleLen {
		errs.WriteValidation(w, errors.New("invalid note"),
			[]string{fmt.Sprintf("title must be between 1 and %d characters", maxNoteTitleLen)})
		return
	}

	var body string
	if req.Body != nil {
		body = *req.Body
	}

	note, err := h.store.CreateNote(r.Context(), gen.CreateNoteParams{
		OwnerID: owner,
		Title:   title,
		Body:    body,
	})
	if err != nil {
		writeNoteStoreError(w, r, err)
		return
	}

	// the content type has to land before WriteHeader; rest.RenderJSON sets it
	// only once the status line is already out.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	rest.RenderJSON(w, toAPINote(note))
}

// Get returns one of the caller's notes. A note owned by someone else is
// reported as missing rather than forbidden, so ids stay unguessable.
func (h *Notes) Get(w http.ResponseWriter, r *http.Request) {
	owner, ok := noteOwner(w, r)
	if !ok {
		return
	}
	id, ok := noteID(w, r)
	if !ok {
		return
	}

	note, err := h.store.GetNote(r.Context(), gen.GetNoteParams{ID: id, OwnerID: owner})
	switch {
	case errors.Is(err, sql.ErrNoRows):
		errs.Write(w, http.StatusNotFound, errors.New("note not found"))
		return
	case err != nil:
		writeNoteStoreError(w, r, err)
		return
	}

	rest.RenderJSON(w, toAPINote(note))
}

// Delete removes one of the caller's notes. It is idempotent: an unknown or
// foreign id deletes nothing and still answers 204.
func (h *Notes) Delete(w http.ResponseWriter, r *http.Request) {
	owner, ok := noteOwner(w, r)
	if !ok {
		return
	}
	id, ok := noteID(w, r)
	if !ok {
		return
	}

	if err := h.store.DeleteNote(r.Context(), gen.DeleteNoteParams{ID: id, OwnerID: owner}); err != nil {
		writeNoteStoreError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// noteOwner resolves the principal subject that owns every row the request may
// touch, writing a 401 when the auth middleware left none behind.
func noteOwner(w http.ResponseWriter, r *http.Request) (string, bool) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		errs.Write(w, http.StatusUnauthorized, errors.New("missing principal"))
		return "", false
	}
	return principal.Subject, true
}

// noteID parses the {id} path parameter, writing a 400 when it is not a uuid.
func noteID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		errs.Write(w, http.StatusBadRequest, errors.New("note id must be a uuid"))
		return uuid.Nil, false
	}
	return id, true
}

// writeNoteStoreError logs the failure and tells the caller only that it failed:
// driver messages are not the caller's business.
func writeNoteStoreError(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "notes store failed", "error", err, "path", r.URL.Path)
	errs.Write(w, http.StatusInternalServerError, errors.New("could not access notes"))
}

func toAPINote(n gen.Note) api.Note {
	return api.Note{
		Id:        n.ID,
		Title:     n.Title,
		Body:      n.Body,
		CreatedAt: n.CreatedAt,
		UpdatedAt: n.UpdatedAt,
	}
}
