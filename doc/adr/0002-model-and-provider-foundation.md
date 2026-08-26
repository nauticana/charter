# 0002: Model and provider foundation

Status: Accepted
Date: 2026-08-25

## Context

The SDK needs typed structs for forty document kinds, a way to resolve references exactly as the schemas define `idRef` (a string resolves in the referring document's namespace), effective-time queries, and provider contracts for eight domains. Downstream applications persist documents in their own stores, so loading must stay pluggable, and Charter must not manage organizations or any other domain.

## Decision

- `model` holds one hand-maintained struct per catalogued kind, grouped by schema domain, and nothing else. A test decodes every Harbor document into its struct, re-encodes it, and validates the result against the schema, so the structs can neither add nor drop normative fields. No generator is used; the round-trip test is the guard.
- `corpus.Source` (`Fetch` by namespace and id, `List` by kind) is the single loading abstraction. `Corpus` is the in-memory implementation; a database is another.
- `corpus.AbstractDocumentProvider` adds kind validation, owner-namespace reference resolution, declared-external handling, and effective-time queries over any `Source`; generic `ResolveAs`, `ListAs`, and `EffectiveAs` return typed documents. Every domain `Base*Provider` is a thin embed of it.
- Effective-time queries use `validity` only: an absent validity is open-ended; lifecycle state is judged by the caller (`organization.Current`, `identity.Lifecycle`).
- Derived views (`organization.Queries`, `process.Graph`, `architecture.Checker`, `evidence.Queries`) depend on the provider interfaces, not on `Corpus`, so a downstream provider reuses them. No relationship is inferred from another: coverage counts direct responsibility assignments only, never role membership or position occupancy.
- Failures are typed sentinels (`corpus.ErrNotFound`, `ErrKindMismatch`, `ErrExternal`) so callers fail closed by kind of failure.

## Consequences

A downstream store implements one small interface and inherits every provider and view. Adding a kind is a struct plus a catalog entry, covered automatically by the round-trip test. Open-ended validity semantics must be kept in mind when a document carries a lifecycle state instead of an end date. The target kinds for an `idRef` property live in schema annotations, as recorded in ADR 0004.

## Specification impact

None. The provider contracts implement `schema/README.md`'s reference semantics as written.
