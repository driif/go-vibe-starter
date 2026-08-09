# CONTEXT.md format

`CONTEXT.md` sits at the repo root and holds two things: the language this project speaks, and a map
of where that language lives in the tree. Nothing else.

## Structure

`make init` writes the two headings and the empty tables. Keep both headings and both table shapes
exactly as they are — `/setup-project` finds the unfilled stub by grepping for the empty row.

```md
# Context

<One or two sentences: what this service is and who it serves.>

## Ubiquitous language

| Term | Means |
|---|---|
| Note | A short piece of text owned by exactly one principal, reached through `/v1/notes`. _Avoid_: memo, post, entry. _DEMO_ |
| Principal | The authenticated caller carried in the request context: a subject plus its roles. Every owner-scoped query filters on its subject. _Avoid_: account, session, identity |
| Realm | The Keycloak tenant that issues and signs the tokens this service accepts. _Avoid_: tenant, org |

## Module map

| Package | Owns |
|---|---|
| `cmd` | cobra commands: `run`, `db migrate`, `db seed` |
| `internal/api/handlers` | HTTP handlers, one file per domain |
| `internal/api/router` | the hub plus one `routes_<domain>.go` per domain |
| `internal/server` | server construction, config, auth, middleware |
| `internal/db/gen` | sqlc output, generated from `sql/queries/` |
| `pkg` | reusable helpers with no project-specific knowledge |
```

## Rules

- **Be opinionated.** When several words exist for one concept, pick one and close the row with
  `_Avoid_: <the rest>`. The rejected words are as useful as the chosen one — they stop the argument
  recurring.
- **Keep definitions to one or two sentences.** Define what the thing *is*, not what the code does
  with it.
- **Only terms specific to this product.** A concept unique to this domain belongs; a general
  programming concept (timeout, middleware, retry) does not, however heavily the project uses it.
- **Split the language table under subheadings** once natural clusters appear. One table is fine
  while the language is small.
- **Keep the map to one row per package**, and only for packages that carry meaning. It answers
  "where does this term live", not "what does this package export" — the code answers that.
- **Mark `DEMO` terms.** Vocabulary belonging to the removable reference slice (`notes`) ends its
  row with `_DEMO_`, so `make init` can strip it along with the code.
