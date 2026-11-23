---
name: go-middleware
description: Adds or changes an HTTP middleware or a Connect interceptor in this repo. Use when a concern has to run on every request or every RPC before the handler - rate limiting, security or cache headers, request-scoped logging or context values, error mapping - or when a ticket asks for a new middleware or interceptor.
---

Read `docs/agents/request-path.md` and `docs/agents/stack-notes.md` first.

1. **Choose the layer.** Concerns needed before decoding, including auth, are HTTP middleware.
   Decoded-message validation and RPC error mapping are Connect interceptors. A domain rule stays
   in that domain. Done when the chosen layer can observe the inputs it needs and reject cheaply.
2. **Implement.** HTTP middleware is `func(http.Handler) http.Handler` in
   `internal/server/middleware/`. Call `next.ServeHTTP` unless deliberately terminating. Writer
   wrappers expose `Unwrap` and preserve Flush via `http.NewResponseController`. Write rejections
   with the Connect error writer and set the result on the request-log carrier. An interceptor is
   `func(next connect.ServerFunc) connect.ServerFunc` in `internal/rpc/interceptor/`; cover unary
   and streaming paths. Done when the concern applies to every intended path.
3. **Register.** Add middleware to root/group `Use` in `internal/server/server.go` before any routes
   on that group, or insert the interceptor at the appropriate position in its chain. No enable flag.
   Done when dependencies appear before consumers in execution order.
4. **Test.** Use the server suite's hand-registered procedures and harness across Connect, JSON,
   gRPC and gRPC-Web. Include pass-through and terminating paths; check streaming flush where relevant.
   Done when `make check` exits 0.
