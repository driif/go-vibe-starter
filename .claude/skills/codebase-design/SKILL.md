---
name: codebase-design
description: Supplies the deep-module vocabulary for Go design in this repo. Use when shaping a package's exported surface, deciding where a seam goes, judging whether a package is deep or shallow, making code testable through its interface, or when another skill needs these words.
---

A deep module offers substantial behavior through a small surface. In Go the package is usually
that module: exported functions, errors, context contracts and concurrency guarantees are its
interface; helper types, SQL and internal ordering are its implementation.

A seam is where an implementation varies without changing callers. Here `server.Deps` injects
runtime dependencies, and a consumer-defined verifier interface accepts Keycloak or local auth.
The in-process issuer tests the real Keycloak adapter. A generated query object is a concrete
implementation; it does not need an interface merely to fake persistence.

Design for locality: a handler maps wire values and calls one store operation. The domain store owns
rules, queries and transaction ordering; it imports no transport package. Avoid handler → service →
repository forwarding chains. `internal/` hides the implementation from other modules.

Keep consumer interfaces small and justified by actual variation. Producers return concrete types.
Use `pgx.BeginFunc` to hide transaction rollback/commit behind the domain operation. Test behavior
through the exported surface using the real local dependencies in `internal/test`.

Before widening a surface, sketch two or three different call sites. Compare what callers must know,
what changes together and where variation lives. Delete pass-through abstractions whose removal
removes complexity rather than spreading it across callers.

For deeper exploration use [DESIGN-IT-TWICE.md](DESIGN-IT-TWICE.md); for dependency classification
and preserving behavior during consolidation use [DEEPENING.md](DEEPENING.md).
