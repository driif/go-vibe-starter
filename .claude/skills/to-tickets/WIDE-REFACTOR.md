# Wide refactors

The exception branch of [`to-tickets`](SKILL.md). Reach it when the work is one mechanical change
whose blast radius fans across the codebase — renaming a column every query selects, changing the
signature every handler calls, moving a type every package imports. A single edit breaks hundreds of
call sites at once, so no tracer bullet can land with `make test` and `make lint` green.

Sequence it **expand–contract** instead. Three phases, each phase its own tickets, every ticket green
on its own because the old form still exists until the last one.

## Expand

One ticket. Add the new form beside the old and leave both working.

- A column: `ALTER TABLE ... ADD COLUMN` in a new goose migration, nullable or defaulted, with the
  old column untouched. Write both in the sqlc queries that populate rows.
- A function or method: add the new signature, and keep the old one as a thin wrapper that calls it.
- An OpenAPI field: add it as optional, so generated clients and `*.gen.go` stay compatible.

Acceptance criteria: the new form exists and is exercised by a test, `make test` and `make lint` exit
0, and no existing caller changed. This ticket blocks every migrate ticket.

## Migrate

One ticket per batch, all blocked by the expand ticket, none blocking each other — so they can run in
any order or in parallel.

Size the batches by the boundary the change actually crosses, usually one package or one domain:
`internal/api/handlers/notes.go` plus its route and test is a batch. Keep each batch's acceptance
criteria to its own call sites, and let `make test` and `make lint` prove the batch. `drift` is the
thing to watch when the change touches generated inputs: run `make gen` inside the batch and check
`git diff --exit-code` before calling it done.

## Contract

One ticket, blocked by every migrate ticket. Delete the old form once no caller remains.

- Drop the column in a goose migration with a working `Down`.
- Delete the wrapper, the deprecated field, the old query.

Acceptance criteria: `grep` for the old name across the tree returns nothing outside `docs/` and
migration history, `make gen` leaves no `drift`, `make test` and `make lint` exit 0.

## When a batch cannot stay green alone

Rare, and worth resisting before accepting. If the call sites genuinely cannot compile independently,
keep the same ticket sequence but land them on a shared integration branch, and add a final
integrate-and-verify ticket blocked by all of them. Green is promised only at that ticket — say so in
its body, so nobody reads a red intermediate ticket as a broken build.
