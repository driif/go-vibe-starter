// DEMO: notes

// Package notes stores notes scoped to their owner's subject.
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

type Note = gen.Note
type Store struct{ queries *gen.Queries }

var ErrNotFound = apperr.New(apperr.NotFound, "NOTE_NOT_FOUND", "note not found")

func NewStore(pool *pgxpool.Pool) *Store { return &Store{gen.New(pool)} }

func (s *Store) Create(ctx context.Context, owner, title, body string) (Note, error) {
	return s.queries.CreateNote(ctx, gen.CreateNoteParams{Owner: owner, Title: title, Body: body})
}
func (s *Store) Get(ctx context.Context, owner string, id uuid.UUID) (Note, error) {
	n, err := s.queries.GetNote(ctx, gen.GetNoteParams{Owner: owner, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return Note{}, ErrNotFound
	}
	return n, err
}
func (s *Store) List(ctx context.Context, owner string) ([]Note, error) {
	return s.queries.ListNotes(ctx, owner)
}
func (s *Store) Delete(ctx context.Context, owner string, id uuid.UUID) error {
	n, err := s.queries.DeleteNote(ctx, gen.DeleteNoteParams{Owner: owner, ID: id})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
