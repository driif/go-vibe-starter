# Domain store

A store holds the pool or generated query object. It owns SQL calls, business rules and transactions.
No transport import belongs in this package.

```go
type Store struct { queries *gen.Queries }

func New(pool *pgxpool.Pool) *Store { return &Store{queries: gen.New(pool)} }

func (s *Store) Get(ctx context.Context, owner string, id uuid.UUID) (gen.Task, error) {
    row, err := s.queries.GetTask(ctx, gen.GetTaskParams{ID: id, Owner: owner})
    if errors.Is(err, pgx.ErrNoRows) {
        return gen.Task{}, apperr.New(apperr.NotFound, "TASK_NOT_FOUND", "task not found")
    }
    if err != nil { return gen.Task{}, fmt.Errorf("get task: %w", err) }
    return row, nil
}
```

Use `pgx.BeginFunc` for a transaction, rebinding queries with `WithTx(tx)`. Error reasons are stable
`<NOUN>_<CONDITION>` strings; messages are safe for callers. A missing foreign-owner row is
not-found, without disclosing whether the row exists. Unexpected persistence errors stay internal.
