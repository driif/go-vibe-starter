---
name: codebase-design
description: The deep-module vocabulary for Go design in this repo. Use when shaping a package's exported surface, deciding where a seam goes, judging whether a package is deep or shallow, making code testable through its interface, or when another skill needs these words.
---

Design **deep modules**: substantial behaviour behind a small interface, sitting at a clean seam,
tested through that interface. Use this vocabulary wherever Go code here is designed or reshaped.
Callers get leverage, maintainers get locality, and the next agent can read one package instead of
five.

## Glossary

Use these words exactly. "Component", "service" and "layer" each name several things at once; one
word meaning one thing across the whole pack is what makes the vocabulary worth carrying.

**Module** — anything with an interface and an implementation. In Go the default module is the
**package**: `internal/server/auth`, `pkg/keycloak`. It scales down to one exported function and up
to `the slice`, but package is where the word usually lands.

**Interface** — everything a caller must know to use the module correctly. The exported identifiers
and their signatures, plus what the zero value does, which errors come back and whether they match
`errors.Is`, what lands in the request context, whether it is safe for concurrent use. Wider than
the `interface` keyword.

**Implementation** — what sits behind that surface: unexported identifiers, helper types, SQL.

**Depth** — leverage at the interface: how much behaviour a caller gets per unit of surface they
must learn. **Deep** is a lot of behaviour behind a small surface. **Shallow** is a surface nearly
as complicated as what it covers.

**Seam** — the place an implementation swaps out without editing the code around it. Here that is a
small consumer-held interface (`auth.TokenVerifier`, `gen.Querier`) or a config switch
(`AUTH_PROVIDER`).

**Adapter** — a concrete type filling a slot at a seam. `*gen.Queries` and a test fake are two
adapters at the `gen.Querier` seam.

**Leverage** — what callers get from depth: capability per unit of surface learned. **Locality** —
what maintainers get: change, bugs and verification concentrate in one package instead of spreading
across call sites.

## How the vocabulary lands in this repo

- **The package is the module.** One directory, one `package` clause, one exported surface. Judge
  depth by reading only the exported identifiers and asking what they buy.
- **Two hiding mechanisms, hiding from different people.** Lowercase identifiers hide from every
  other package in this module — the one you reach for daily, and usually the whole of a deepening.
  `internal/` hides from anyone importing the module, which is why `pkg/` — `db`, `dotenv`,
  `keycloak`, `oidc`, `tests` — is the whole surface a project built from this template inherits.
- **Interfaces are defined at the consumer and kept to 1–3 methods.** `internal/server/auth`
  declares `TokenVerifier` with one method because `Authenticate` is what needs it; `pkg/keycloak`
  returns a concrete `*Verifier` and declares no interface at all. Producers return concrete types.
- **`gen.Querier` is a real seam.** sqlc generates it, and two adapters justify it: `*gen.Queries`
  against Postgres, a fake in a handler test. A handler that takes `gen.Querier` is testable without
  Docker.
- **The shallow tell: a handler that reaches through three packages to do its job.** When a handler
  in `internal/api/handlers` pulls `s.KeycloakAdmin`, `s.DB` and `s.Config` and assembles the
  behaviour itself, the packages it called offered no depth, so the caller supplied it — and the
  next handler will supply it again. Deepening means giving that work a home: one exported call in
  one package, invoked once.

## Deep vs shallow, in Go

```go
// shallow — five exported calls, and every caller repeats the same order
in, err := notes.Validate(body)
tx, err := notes.Begin(ctx, db)
row, err := notes.Insert(ctx, tx, ownerID, in)
err = notes.Touch(ctx, tx, row.ID)
err = notes.Commit(tx)

// deep — one call; validation, ordering and rollback live behind it
note, err := notes.Create(ctx, ownerID, in)
```

Designing a surface, ask: can I export fewer identifiers? Can I simplify the parameters (`ctx`
first, then one params struct, the shape sqlc already generates)? Can I hide more inside?

## Principles

- **Depth is a property of the interface, not the implementation.** A deep package can be built
  internally from small swappable parts — they just stay unexported. Internal seams are welcome;
  keep them off the exported surface even when tests use them.
- **The deletion test.** Delete the package in your head. If the complexity vanishes, it was a
  pass-through. If it reappears at every call site, it was earning its keep.
- **The interface is the test surface.** Tests and callers cross the same seam. Wanting to test past
  the exported surface says the package is the wrong shape, not that the test needs more access.
- **One adapter is a hypothetical seam; two make it real.** Production plus a test fake counts as
  two. This is the `AGENTS.md` interface rule read as a design test: an interface with one
  implementation is indirection wearing a seam's clothes.

## Testability follows depth

1. **Accept dependencies, don't build them.** A function taking `gen.Querier` is testable; one
   calling `sql.Open` itself is not.
2. **Return the result, let the caller cause the effect.** `Render(note) ([]byte, error)` tests in
   one line; a function that writes to the `http.ResponseWriter` mid-computation needs a recorder
   and a decoder before it can be asserted on.
3. **Keep the parameter list short.** Fewer exported identifiers means fewer tests; fewer parameters
   means less setup per test.

## Design it twice

Your first interface is rarely the best one. Before writing the implementation, sketch **two or
three radically different** surfaces for the same behaviour — one minimal (1–3 entry points), one
optimised for the most common caller, one that puts the seam somewhere else entirely — and compare
them on depth, locality and seam placement. Sketch the **call site**, not the body: the design shows
up in what the caller has to write. Then pick one and say why in a sentence.

Exploring further than a sketch — briefs, parallel exploration, a full comparison — see
[`DESIGN-IT-TWICE.md`](DESIGN-IT-TWICE.md). Deepening a cluster of shallow packages given what they
depend on, and what happens to their tests, see [`DEEPENING.md`](DEEPENING.md).
