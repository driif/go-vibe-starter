# Request path

HTTP middleware handles concerns needed before message decoding. Connect interceptors handle decoded
messages and RPC errors. Authentication belongs at HTTP level, validation and domain-error mapping
at RPC level. See `internal/server/server.go` for the root chain and group registration order.

Public routes are `/healthz` and `/readyz`. RPC services mount on the authenticated group; a public
RPC requires a deliberately separate server and mount. Auth rejects before reading the request body.
The error writer speaks Connect, gRPC and gRPC-Web. A Connect GET URL needs the protocol's query
parameters; a raw browser navigation is not a valid RPC.

```text
HTTP → request id / access log / metrics / security / recovery / CORS
     → public probes
     → bearer authentication → Connect decoding → error mapping / request logging / validation
       → handler → domain store → owner-scoped SQL
```

Missing or invalid bearer tokens yield `unauthenticated` (HTTP 401). Provider transport failure
yields `unavailable` (503). A valid caller without a required role gets `permission_denied` (403).
A missing/foreign row returns `not_found`. CORS preflight does not require a token. Protocol errors
and auth rejections may never reach an interceptor; HTTP metrics and logging still record them.

```sh
curl -fsS -H 'Content-Type: application/json' -d '{}' http://localhost:8080/api.v1.WhoAmIService/WhoAmI
curl -fsS 'http://localhost:8080/api.v1.WhoAmIService/WhoAmI?connect=v1&encoding=json&message=%7B%7D'
```

For local Keycloak run `make keycloak-up`, set the auth provider in `.env.local` to `keycloak`,
then `make -s token` for an access token. Add `Authorization: Bearer <token>` to the request.
