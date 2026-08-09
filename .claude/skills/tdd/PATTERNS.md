# Test patterns

Worked shapes for [`tdd`](SKILL.md). Each one is the form this repo already uses — copy the shape,
not the subject.

## Which package a test file declares

- `package <same>` when the test needs an unexported seam: `internal/api/handlers` tests build
  `&Notes{store: fake}` directly, which only an in-package test can do.
- `package <same>_test` when the behaviour is fully visible from outside, as
  `internal/server/config/auth_config_test.go` does. It forces you to test through the exported
  surface, which is the point.

## Table-driven

The canonical example lives in `internal/server/config/auth_config_test.go`. The shape:

```go
func TestAppValidate(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantErr  string
	}{
		{name: "keycloak accepted", provider: config.AuthProviderKeycloak},
		{name: "unknown provider names the accepted values", provider: "oauth2", wantErr: "accepted values"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultServiceConfigFromEnv() // fresh per case
			cfg.Auth.Provider = tt.provider

			err := cfg.Validate()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
```

`cfg` is built inside the subtest, so no case can see another's mutation. Environment-driven cases
use `t.Setenv`, which restores the previous value when the subtest ends.

## Handler, with the store faked

`internal/api/handlers/notes_test.go` is the canonical example. Two moves make it work:

**The store seam.** The handler declares `noteStore`, the slice of the sqlc querier it uses, so the
test substitutes an in-memory map — no Docker, and error paths become one field:

```go
type fakeNoteStore struct {
	notes map[uuid.UUID]gen.Note
	// err, when set, fails every call the way a dead database would.
	err error
}
```

**The real route.** Mount the handlers through chi exactly as `registerNotes` does, so `chi.URLParam`
resolves and the `principal` arrives the way production supplies it — through the middleware, since
the context key is unexported:

```go
func newNotesRouter(t *testing.T, store noteStore, subject string) http.Handler {
	t.Helper()

	h := &Notes{store: store}
	r := chi.NewRouter()
	r.Use(auth.DevPrincipal(subject, nil))
	r.Get(notesPath, h.List)
	r.Get(notesPath+"/{id}", h.Get)
	return r
}
```

Each table case then builds a fresh store, serves one request through that router, and asserts on the
status and the decoded body — what the caller observes. Ownership is proved by asking for another
owner's id and requiring a 404, not by inspecting the fake.

## Against a real Postgres

`internal/server/test.E2e` starts a testcontainers Postgres, runs the goose migrations, builds the
server and registers every route, then hands you the `*server.Server`. It needs a Docker daemon, so
gate it:

```go
func TestNotesRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("needs docker")
	}

	test.E2e(t, func(s *server.Server) {
		rec := httptest.NewRecorder()
		s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/notes", nil))
		require.Equal(t, http.StatusOK, rec.Code)
	})
}
```

Reach for this when the SQL is what is under test — the `owner_id` scoping against real rows, a
constraint, a migration's default. For branch coverage of the handler, the faked store above is
faster and deterministic.

## Filesystem and helpers

`t.TempDir()` gives a per-test directory that is removed automatically; nothing else should write to
disk. Any helper that asserts or fatals starts with `t.Helper()`, so failures report the caller's
line:

```go
func decodeNote(t *testing.T, body io.Reader) api.Note {
	t.Helper()
	var note api.Note
	require.NoError(t, json.NewDecoder(body).Decode(&note))
	return note
}
```
