# Slice tests

Use `internal/test` in an external test package. Docker is required. Build the production stack
with `test.NewServer`, create a generated client using `s.Client(s.Token(...))`, and observe
responses over HTTP. Read the permanent `internal/rpc/whoami/whoami_test.go` for exact signatures.

Cases cover success, each error reason, a foreign owner's row, a rejected input and no token.
Assert `connect.CodeOf(err)`; inspect details with `test.Reason` and `test.Violations` where supplied
by the harness. A foreign-owner lookup must be not-found. Store tests use `test.Database(t)`.
Use table cases, `t.Parallel()` and fresh fixtures. Finish with `make check`.
