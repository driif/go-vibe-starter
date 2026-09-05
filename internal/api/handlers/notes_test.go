// DEMO: reference vertical slice — delete with `make init` or by hand.

package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/driif/go-vibe-starter/internal/api"
	"github.com/driif/go-vibe-starter/internal/db/gen"
	"github.com/driif/go-vibe-starter/internal/server/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	notesOwner      = "owner-1"
	notesOtherOwner = "owner-2"
	notesPath       = "/v1/notes"
)

var errNotesStoreDown = errors.New("connection refused")

// fakeNoteStore is an in-memory noteStore. The handlers only need rows keyed by
// id and filtered by owner, so a map beats a real database here and keeps the
// test free of Docker.
type fakeNoteStore struct {
	notes map[uuid.UUID]gen.Note
	// err, when set, fails every call the way a dead database would.
	err error
}

func newFakeNoteStore(notes ...gen.Note) *fakeNoteStore {
	s := &fakeNoteStore{notes: make(map[uuid.UUID]gen.Note, len(notes))}
	for _, n := range notes {
		s.notes[n.ID] = n
	}
	return s
}

func (s *fakeNoteStore) CreateNote(_ context.Context, arg gen.CreateNoteParams) (gen.Note, error) {
	if s.err != nil {
		return gen.Note{}, s.err
	}
	note := gen.Note{
		ID:        uuid.New(),
		OwnerID:   arg.OwnerID,
		Title:     arg.Title,
		Body:      arg.Body,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	s.notes[note.ID] = note
	return note, nil
}

func (s *fakeNoteStore) GetNote(_ context.Context, arg gen.GetNoteParams) (gen.Note, error) {
	if s.err != nil {
		return gen.Note{}, s.err
	}
	note, ok := s.notes[arg.ID]
	if !ok || note.OwnerID != arg.OwnerID {
		return gen.Note{}, sql.ErrNoRows
	}
	return note, nil
}

func (s *fakeNoteStore) ListNotesByOwner(_ context.Context, ownerID string) ([]gen.Note, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := []gen.Note{}
	for _, n := range s.notes {
		if n.OwnerID == ownerID {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *fakeNoteStore) DeleteNote(_ context.Context, arg gen.DeleteNoteParams) error {
	if s.err != nil {
		return s.err
	}
	if note, ok := s.notes[arg.ID]; ok && note.OwnerID == arg.OwnerID {
		delete(s.notes, arg.ID)
	}
	return nil
}

// newNotesRouter mirrors registerNotes so the tests exercise the real path
// parameters and the principal the middleware injects.
func newNotesRouter(t *testing.T, store noteStore, subject string) http.Handler {
	t.Helper()

	h := &Notes{store: store}
	r := chi.NewRouter()
	r.Use(auth.DevPrincipal(subject, nil))
	r.Get(notesPath, h.List)
	r.Post(notesPath, h.Create)
	r.Get(notesPath+"/{id}", h.Get)
	r.Delete(notesPath+"/{id}", h.Delete)
	return r
}

func doNotes(t *testing.T, h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func newNote(t *testing.T, owner, title string, createdAt time.Time) gen.Note {
	t.Helper()

	return gen.Note{
		ID:        uuid.New(),
		OwnerID:   owner,
		Title:     title,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func TestNotesCreate(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
		wantTitle  string
		wantBody   string
	}{
		{
			name:       "title and body",
			body:       `{"title":"Shopping list","body":"milk, bread"}`,
			wantStatus: http.StatusCreated,
			wantTitle:  "Shopping list",
			wantBody:   "milk, bread",
		},
		{
			name:       "body defaults to empty",
			body:       `{"title":"Shopping list"}`,
			wantStatus: http.StatusCreated,
			wantTitle:  "Shopping list",
		},
		{
			name:       "title is trimmed",
			body:       `{"title":"  Shopping list  "}`,
			wantStatus: http.StatusCreated,
			wantTitle:  "Shopping list",
		},
		{
			name:       "blank title rejected",
			body:       `{"title":"   "}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing title rejected",
			body:       `{"body":"milk"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "overlong title rejected",
			body:       `{"title":"` + strings.Repeat("a", maxNoteTitleLen+1) + `"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "malformed json rejected",
			body:       `{"title":`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "store failure is a 500",
			body:       `{"title":"Shopping list"}`,
			storeErr:   errNotesStoreDown,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeNoteStore()
			store.err = tt.storeErr

			rec := doNotes(t, newNotesRouter(t, store, notesOwner), http.MethodPost, notesPath, tt.body)
			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())

			if tt.wantStatus != http.StatusCreated {
				assert.Empty(t, store.notes)
				return
			}

			var got api.Note
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			assert.Equal(t, tt.wantTitle, got.Title)
			assert.Equal(t, tt.wantBody, got.Body)
			assert.NotEqual(t, uuid.Nil, got.Id)

			stored, ok := store.notes[got.Id]
			require.True(t, ok, "note was not persisted")
			assert.Equal(t, notesOwner, stored.OwnerID, "the note must be owned by the caller")
		})
	}
}

func TestNotesGet(t *testing.T) {
	mine := newNote(t, notesOwner, "mine", time.Now())
	theirs := newNote(t, notesOtherOwner, "theirs", time.Now())

	tests := []struct {
		name       string
		id         string
		storeErr   error
		wantStatus int
	}{
		{name: "own note", id: mine.ID.String(), wantStatus: http.StatusOK},
		{name: "another owner's note is missing", id: theirs.ID.String(), wantStatus: http.StatusNotFound},
		{name: "unknown id", id: uuid.NewString(), wantStatus: http.StatusNotFound},
		{name: "non-uuid id", id: "not-a-uuid", wantStatus: http.StatusBadRequest},
		{name: "store failure is a 500", id: mine.ID.String(), storeErr: errNotesStoreDown, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeNoteStore(mine, theirs)
			store.err = tt.storeErr

			rec := doNotes(t, newNotesRouter(t, store, notesOwner), http.MethodGet, notesPath+"/"+tt.id, "")
			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())

			if tt.wantStatus != http.StatusOK {
				return
			}

			var got api.Note
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			assert.Equal(t, mine.ID, got.Id)
			assert.Equal(t, "mine", got.Title)
		})
	}
}

func TestNotesList(t *testing.T) {
	now := time.Now()
	older := newNote(t, notesOwner, "older", now.Add(-time.Hour))
	newer := newNote(t, notesOwner, "newer", now)
	theirs := newNote(t, notesOtherOwner, "theirs", now)

	tests := []struct {
		name       string
		stored     []gen.Note
		storeErr   error
		wantStatus int
		wantTitles []string
	}{
		{
			name:       "no notes yields an empty array",
			wantStatus: http.StatusOK,
			wantTitles: []string{},
		},
		{
			name:       "newest first, other owners excluded",
			stored:     []gen.Note{older, newer, theirs},
			wantStatus: http.StatusOK,
			wantTitles: []string{"newer", "older"},
		},
		{
			name:       "store failure is a 500",
			stored:     []gen.Note{newer},
			storeErr:   errNotesStoreDown,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeNoteStore(tt.stored...)
			store.err = tt.storeErr

			rec := doNotes(t, newNotesRouter(t, store, notesOwner), http.MethodGet, notesPath, "")
			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())

			if tt.wantStatus != http.StatusOK {
				return
			}

			var got []api.Note
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

			titles := make([]string, len(got))
			for i, n := range got {
				titles[i] = n.Title
			}
			assert.Equal(t, tt.wantTitles, titles)
		})
	}
}

func TestNotesDelete(t *testing.T) {
	mine := newNote(t, notesOwner, "mine", time.Now())
	theirs := newNote(t, notesOtherOwner, "theirs", time.Now())

	tests := []struct {
		name       string
		id         string
		storeErr   error
		wantStatus int
		wantLeft   int
	}{
		{name: "own note", id: mine.ID.String(), wantStatus: http.StatusNoContent, wantLeft: 1},
		{name: "another owner's note survives", id: theirs.ID.String(), wantStatus: http.StatusNoContent, wantLeft: 2},
		{name: "unknown id is still a 204", id: uuid.NewString(), wantStatus: http.StatusNoContent, wantLeft: 2},
		{name: "non-uuid id", id: "not-a-uuid", wantStatus: http.StatusBadRequest, wantLeft: 2},
		{name: "store failure is a 500", id: mine.ID.String(), storeErr: errNotesStoreDown, wantStatus: http.StatusInternalServerError, wantLeft: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeNoteStore(mine, theirs)
			store.err = tt.storeErr

			rec := doNotes(t, newNotesRouter(t, store, notesOwner), http.MethodDelete, notesPath+"/"+tt.id, "")
			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			assert.Len(t, store.notes, tt.wantLeft)
		})
	}
}

// TestNotesWithoutPrincipal covers the handlers being mounted outside the auth
// group by mistake: they must refuse rather than fall back to an empty owner.
func TestNotesWithoutPrincipal(t *testing.T) {
	h := &Notes{store: newFakeNoteStore()}
	r := chi.NewRouter()
	r.Get(notesPath, h.List)
	r.Post(notesPath, h.Create)
	r.Get(notesPath+"/{id}", h.Get)
	r.Delete(notesPath+"/{id}", h.Delete)

	tests := []struct {
		method string
		target string
	}{
		{http.MethodGet, notesPath},
		{http.MethodPost, notesPath},
		{http.MethodGet, notesPath + "/" + uuid.NewString()},
		{http.MethodDelete, notesPath + "/" + uuid.NewString()},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.target, func(t *testing.T) {
			rec := doNotes(t, r, tt.method, tt.target, `{"title":"Shopping list"}`)
			assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
		})
	}
}
