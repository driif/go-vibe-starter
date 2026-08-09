# Deepening

How to turn a cluster of shallow packages into one deep module without breaking the tests that
guard it. Assumes the vocabulary in [`SKILL.md`](SKILL.md) — module, interface, seam, adapter.

## Classify the dependencies first

What the cluster depends on decides how the deepened module gets tested across its seam.

**1. In-process.** Pure computation and in-memory state: validation, mapping a `gen.Note` to an API
type, building a query filter. Always deepenable — merge the packages, test through the new exported
surface, no seam needed.

**2. Local-substitutable.** A real dependency with a stand-in that runs inside `go test`: Postgres
through the helpers in `internal/server/test`, the filesystem through `t.TempDir()`, an HTTP
dependency through `httptest.Server`. Deepenable as soon as the stand-in exists. The seam stays
internal — the deepened package's exported surface says nothing about it.

**3. Remote but owned.** Another service you also ship. None in this template today. When one
appears: declare the interface in the consumer, inject an HTTP adapter in production and an
in-memory adapter in tests, and keep the logic in one deep module even though it deploys across a
network.

**4. True external.** A third party you do not control — Keycloak here. `internal/server/auth` is
the worked example: one-method `TokenVerifier` declared at the consumer, `*keycloak.Verifier` as the
production adapter, a fake as the test adapter, and `AUTH_PROVIDER=none` swapping the whole
behaviour for a static dev `principal`.

## Seam discipline

- **One adapter is a hypothetical seam; two make it real.** Production plus a test adapter counts as
  two — that is what earns `gen.Querier` and `auth.TokenVerifier` their place. Below that bar, take
  the concrete type.
- **Internal seams stay internal.** A deep package may split itself on unexported interfaces for its
  own tests. Exporting them because a test found them handy widens the interface and undoes the
  depth you just bought.
- **Put the seam where variation actually lives.** `AUTH_PROVIDER` is a seam because two providers
  ship. A `StorageProvider` interface with one Postgres implementation is not.

## Replace tests, don't layer them

- Write the new tests at the deepened module's exported surface first, and watch them fail before
  the merge is finished — `red-green`, from the `tdd` skill, applies to a refactor too.
- Delete the old tests that poked at the shallow pieces. Once behaviour is covered through the new
  surface they are duplicate coverage that pins the old shape in place.
- Assert observable outcomes through the interface: returned values, status codes, rows in Postgres.
  A test that survives an internal rewrite is testing the interface; one that has to change is
  testing past it.
- `make test` and `make lint` both exit 0 before the deepening is called done, and the diff deletes
  more lines than it adds — if it does not, check whether anything was actually hidden.
