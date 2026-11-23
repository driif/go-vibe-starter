# Test patterns

Use external test packages when the public API expresses the behavior. Pure functions may use an
in-package test for private details. Integration tests import `internal/test`, which imports the
production server, so they must be external to avoid cycles.

## Store

Obtain `test.Database(t)` and build the real store on that pool. Assert the stored value and the
foreign-owner result. Use fresh fixtures per subtest; Postgres databases are isolated clones.

## Handler

Use `test.NewServer(t)`, mint a token through its issuer and call through the generated client.
The permanent `internal/rpc/whoami/whoami_test.go` is the signature reference. Run cases for success,
no token, each stable error reason, rejected inputs and ownership. Exercise the four protocol
configurations where behavior lives below a particular RPC.

## Middleware and interceptors

The server suite registers unary and streaming test procedures directly. Reuse that shape to test
an HTTP concern over the real stack. A response wrapper must continue to flush streams.

## Pure functions and filesystem

Use table-driven cases with independent expected values, testify require/assert, `t.Helper()` in
helpers and `t.TempDir()` for files. Environment cases use `t.Setenv` and cannot run in parallel.
Docker is a prerequisite for integration tests. Finish with `make check`.
