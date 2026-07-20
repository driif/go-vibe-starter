---
name: add-endpoint
description: Use when adding an HTTP endpoint — OpenAPI-first: edit oapi/openapi.yaml, run make gen-oapi, implement the handler, and register the route with the right auth.
---

# Add endpoint

This repo is OpenAPI-first. Ask for the method, path, domain (e.g. `users`), and whether it needs
authentication/roles, if not given. Then:

## 1. Describe it in the spec

Add the path, operation, and request/response schemas to `oapi/openapi.yaml`.

## 2. Regenerate

```bash
make gen-oapi
```

This rewrites `internal/api/openapi_types.gen.go` and `internal/api/openapi_server.gen.go`. Never
hand-edit generated files.

## 3. Implement the handler

In `internal/api/handlers/<domain>.go` (package `handlers`). Use a plain handler when it needs
nothing from the server, or a closure when it needs server dependencies:

```go
// plain
func GetThing(w http.ResponseWriter, r *http.Request) {
	// ...
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// closure with server deps
func ListThings(s *server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.SomeDep == nil {
			errs.Write(w, http.StatusServiceUnavailable, fmt.Errorf("dep not configured"))
			return
		}
		// ...
	}
}
```

- Read path params with `chi.URLParam(r, "id")`; query params with `r.URL.Query().Get(...)`.
- Write errors with `errs.Write(w, status, err)` (see `internal/server/errs`).
- Log with `log/slog`.

## 4. Register the route

In `internal/api/router/routes.go` → `RegisterHandlersV1`, inside the group, with the right auth:

```go
r.Get("/v1/things/me", handlers.GetThing)                                    // authenticated (group applies Authenticate)
r.With(auth.RequireRealmRoles(false, "admin")).Get("/v1/things", handlers.ListThings(s)) // admin-only
```

Match the existing auth pattern: the group already applies `auth.Authenticate`; add
`auth.RequireRealmRoles`/`auth.RequireClientRoles` for role-gated routes. An admin-only listing
endpoint must not be reachable with a plain bearer token.

## Fallback

For a quick internal route that isn't part of the public contract, a raw chi handler is acceptable,
but prefer the codegen path above. Don't add error-wrapping layers without a clear benefit.
