# Stack notes

Use the versions in `go.mod`, the local module source and generated code as the API reference.

- Connect v2 uses `connectrpc.com/connect/v2`, `connect.ServerFunc`, a server registration API,
  and generated service helpers. Its unary methods take message pointers, unlike older Request/Response
  wrappers. Read `internal/gen/api/v1/apiv1connect/` before writing a handler or client.
- routegroup wraps stdlib ServeMux patterns. Call `Use` before registering any route on that group.
  Mount a Connect service on its generated prefix without method qualification. A method-qualified
  catch-all can conflict with those mounts. Plain HTTP path values use `r.PathValue`.
- A response writer wrapper exposes `Unwrap` and preserves `Flush` through
  `http.NewResponseController`; do not wrap a writer with a unary-only logger.
- pgx uses `pgx.ErrNoRows`, explicit types and named SQL parameters in sqlc sources. It does not
  implicitly encode an integer as text. Nullable values use `pgtype`.
- goose uses the pgx stdlib adapter at the migration boundary. The rest of persistence uses pgx pools.
- `buf generate` deletes everything under `internal/gen`. Keep handwritten code elsewhere.
