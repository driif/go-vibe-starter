# Environment reference

The process environment takes precedence over `.env.local` in the working directory, then
built-in defaults. Empty values count as unset. Tests ignore `.env.local`. `make run` and the
migration targets create it from `.env.example` if absent. Parsing reports malformed values;
`run` also validates required configuration. Connection strings are never printed in errors.

| Variable | Built-in default | Meaning |
|---|---|---|
| `APP_ENVIRONMENT` | `production` | `development`, `dev`, `local`, `test` permit local auth; case and surrounding whitespace are normalized |
| `AUTH_PROVIDER` | none; required | `keycloak` or `none` |
| `AUTH_DEV_ROLES` | empty | comma-separated realm roles for the `local-dev` principal |
| `KEYCLOAK_ISSUER_URL` | empty | required with Keycloak; absolute HTTP(S) issuer, no query, fragment or trailing slash |
| `KEYCLOAK_AUDIENCE` | empty | required with Keycloak; access-token audience |
| `DATABASE_URL` | none; required | Postgres URL; pgx pool options may be query parameters |
| `LISTEN_ADDR` | `:8080` | public HTTP listen address |
| `MANAGEMENT_ADDR` | empty; disabled | separate private metrics and pprof listener |
| `CORS_ALLOWED_ORIGINS` | empty; no cross-origin access | comma-separated exact HTTP(S) origins, no wildcard or path |
| `LOG_LEVEL` | `INFO` | `DEBUG`, `INFO`, `WARN`, `ERROR` |

Local compose values live in `.env.example`, which explicitly adds the admin dev role and binds
both listeners to loopback. Deployment supplies its own database URL and
Keycloak issuer/audience. Auth without verification is refused outside the development allow-list.
