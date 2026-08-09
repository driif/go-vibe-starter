# Tests (step 5 of the slice)

The `tdd` skill owns **red-green**, the shape of a Go test here, and the worked patterns — the fake
store, the chi router a handler test mounts, the testcontainers gate. Invoke it and copy from there.

This file is the slice-specific part: which layer carries which test, and what "done" means for a
feature.

| Layer | Test | Needs Docker |
|---|---|---|
| handler | fake store + `httptest`, one case per status in the spec | no |
| middleware | wrap a recording handler, assert what reached it | no |
| query + migration | real postgres through `internal/server/test` | yes |
| wiring | `test.E2e` — server, router and database together | yes |

Handler tests carry the weight: they are fast, they need nothing running, and they cover the branch
that matters most, which is ownership. Reach for the database layers when the SQL itself is the
thing in doubt — the `owner_id` scoping against real rows, a unique constraint, a migration's
default.

## What the handler test must cover

Work through the operation's `responses` map in `oapi/openapi.yaml` and give each status a row. The
map is the checklist: a status listed there with no case is a status nobody tests.

The foreign-owner row is not optional — a second subject asking for the first subject's id must get
404, not the row. That case is the only proof the owner scoping in the SQL is actually reached, so
keep the fake's `OwnerID` matching as strict as the `WHERE` clause it stands in for.

For a route the router gates by role, wrap it in `auth.RequireRealmRoles(...)` in the test router
too, and add the 403 row.

Assert the response body where the shape is the point — decode into the generated `api.*` type and
compare fields, rather than matching raw JSON text.

## The import cycle to avoid

`internal/server/test` imports `internal/api/router`, which imports `internal/api/handlers`. A test
in `package handlers` that imports it is an import cycle. Put database-backed tests for a domain in
the external `package handlers_test`, or in a package of their own.

## Done

`make test` exits 0, every status in the spec has a case, and the foreign-owner case is one of them.
