# Handler and mount

Read `internal/rpc/whoami/` and `internal/server/routes_whoami.go` for the current Connect v2 API.
A handler maps request → principal → store → response. It contains no SQL or business rules.
Use `identity.FromContext(ctx)` and pass the subject as owner. Role checks are explicit:

```go
if !identity.FromContext(ctx).HasRole("admin") {
    return nil, apperr.New(apperr.PermissionDenied, "ADMIN_REQUIRED", "admin role required")
}
```

Implement the whole generated handler interface; do not embed an unimplemented handler. Each domain
gets a mount file with the same signature as the permanent example; the hub calls that mount once.
Register it on the authenticated group. A deliberately public service needs a separate Connect
server on the public group so the protected service cannot accidentally become public.

The template ships no streaming service. Auth checks once at stream start; the server adds no
wall-time bound and shutdown cuts an open stream after the grace period. Follow
`docs/production.md` when introducing streams. Finish with `make check`.
