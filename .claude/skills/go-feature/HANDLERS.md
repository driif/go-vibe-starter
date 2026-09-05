# Handlers and routes (step 4 of the slice)

Handlers live in `internal/api/handlers/<domain>.go`, routes in
`internal/api/router/routes_<domain>.go`. Worked example: `internal/api/handlers/notes.go` and
`internal/api/router/routes_notes.go`.

## Pick the shape

Three forms are in use; take the narrowest one that carries the dependencies the handler needs.

| Dependencies | Form | Example |
|---|---|---|
| none | plain `func(w, r)` | `handlers.GetMe`, `handlers.Health` |
| something off the server | closure `func(s *server.Server) http.HandlerFunc` | `handlers.Ready(s)`, `handlers.ListUsers(s)` |
| the database | struct + constructor, methods as handlers | `handlers.NewNotes(s.DB)` |

Anything touching the database takes the struct form, because that is where the **seam** goes:

```go
// noteStore is the slice of the sqlc-generated querier these handlers use.
// Declared at the consumer, so a test can substitute a fake without a database.
type noteStore interface {
	CreateNote(ctx context.Context, arg gen.CreateNoteParams) (gen.Note, error)
	GetNote(ctx context.Context, arg gen.GetNoteParams) (gen.Note, error)
	ListNotesByOwner(ctx context.Context, ownerID string) ([]gen.Note, error)
	DeleteNote(ctx context.Context, arg gen.DeleteNoteParams) error
}

type Notes struct {
	store noteStore
}

// NewNotes builds the notes handlers over the generated queries.
func NewNotes(db gen.DBTX) *Notes {
	return &Notes{store: gen.New(db)}
}
```

`*gen.Queries` satisfies `gen.Querier` and therefore this narrower interface. Declaring the handful
of methods the domain uses — rather than depending on `gen.Querier` whole — keeps the fake in the
test file the same size as the interface.

## Inside a handler

Order is fixed: resolve the caller, read and validate input, call the store, render. Every failure
returns early.

```go
func (h *Notes) Get(w http.ResponseWriter, r *http.Request) {
	owner, ok := noteOwner(w, r)      // 401 written inside when absent
	if !ok {
		return
	}
	id, ok := noteID(w, r)            // 400 written inside when not a uuid
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
```

- **Principal.** `auth.PrincipalFromContext(r.Context())` returns `(*keycloak.Principal, bool)`;
  `principal.Subject` is the owner id. A missing principal is a 401, never a fallback to an
  unscoped query. Roles are `principal.HasRealmRole(...)`, but role gating belongs on the route.
- **Owner scoping.** Pass the subject into every query as the owner argument. A caller asking for
  another owner's id gets `sql.ErrNoRows` from the SQL itself, which renders as 404 — the id stays
  unguessable, and there is no ownership `if` to forget.
- **Path params.** `chi.URLParam(r, "id")`, then parse: `uuid.Parse` for a uuid path param.
  Query params come from `r.URL.Query().Get("...")`.
- **Body.** `json.NewDecoder(r.Body).Decode(&req)` into the generated `api.*` request type, then
  validate: trim, check length and range, and reject with `errs.WriteValidation(w, err, []string{…})`
  listing what is wrong. `oapi-codegen` generates no validation, so constraints in the spec are
  enforced here.
- **Errors.** `errs.Write(w, status, err)` for a single problem, `errs.WriteValidation` for a list.
  A store failure is logged with `slog.ErrorContext` and rendered as a 500 with a generic message —
  the driver's text is not the caller's business.
- **Success.** `rest.RenderJSON(w, payload)` for 200. For any other success status set
  `Content-Type` and call `w.WriteHeader(status)` first, because `rest.RenderJSON` sets the header
  too late once the status line is out. 204 is `w.WriteHeader(http.StatusNoContent)` and no body.
- Map the row to the wire type in one small function (`toAPINote`), keeping `gen.Note` out of the
  response.

## Register the route

```go
func registerNotes(s *server.Server) {
	notes := handlers.NewNotes(s.DB)

	s.Router.Group(func(r chi.Router) {
		r.Use(s.Authenticator())
		r.Get("/v1/notes", notes.List)
		r.Post("/v1/notes", notes.Create)
		r.Get("/v1/notes/{id}", notes.Get)
		r.Delete("/v1/notes/{id}", notes.Delete)
	})
}
```

- `s.Authenticator()` is the authentication seam: real token verification under
  `AUTH_PROVIDER=keycloak`, a static dev principal under `none`. Handlers see the same context
  either way.
- Role-gate a single route with `r.With(auth.RequireRealmRoles(false, "admin")).Get(...)`, as
  `routes_users.go` does. Other gates: `RequireClientRoles`, `RequireScopes`,
  `RequireOrganization`, `RequireAnyOrganization`.
- Routes that must answer without a token stay outside the group — see `routes_health.go`.
- Add `register<Domain>(s)` to `RegisterHandlersV1` in the **hub**,
  `internal/api/router/routes.go`. One file per domain is what keeps adding a domain from editing a
  shared file.
