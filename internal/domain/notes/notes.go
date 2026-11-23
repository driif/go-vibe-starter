// DEMO: notes

package notes

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/driif/go-vibe-starter/internal/apperr"
	"github.com/driif/go-vibe-starter/internal/db/gen"
)

// Note is a stored note.
type Note = gen.Note

// ErrNotFound is returned for a missing note and for another owner's note.
var ErrNotFound = apperr.New(apperr.NotFound, "NOTE_NOT_FOUND", "note not found")

// Store reads and writes notes, always scoped by owner.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns a Store over pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Create stores a note of owner.
func (s *Store) Create(ctx context.Context, owner, title, body string) (Note, error) {
	return gen.New(s.pool).CreateNote(ctx, gen.CreateNoteParams{Owner: owner, Title: title, Body: body})
}

// Get returns the note id of owner.
func (s *Store) Get(ctx context.Context, owner string, id uuid.UUID) (Note, error) {
	note, err := gen.New(s.pool).GetNote(ctx, gen.GetNoteParams{ID: id, Owner: owner})
	if errors.Is(err, pgx.ErrNoRows) {
		return Note{}, ErrNotFound
	}
	return note, err
}

// List returns the newest 100 notes of owner, newest first.
func (s *Store) List(ctx context.Context, owner string) ([]Note, error) {
	return gen.New(s.pool).ListNotes(ctx, owner)
}

// Delete removes the note id of owner.
func (s *Store) Delete(ctx context.Context, owner string, id uuid.UUID) error {
	n, err := gen.New(s.pool).DeleteNote(ctx, gen.DeleteNoteParams{ID: id, Owner: owner})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
