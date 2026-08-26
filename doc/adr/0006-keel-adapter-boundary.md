# 0006: Keel adapter boundary

Status: Proposed
Date: 2026-08-25

## Context

Keel (v1.2.54) is the backend framework beneath the reference runtime. Its review concluded that keel already exposes the primitives a Charter adapter needs (principal and tenant context keys, `CheckActionPermission`, trust guards, the table change logger, message publishing, metrics, request ids) and that everything Charter-shaped belongs in Charter or an adapter. Charter's core must stay platform-free, and keel must never depend on Charter.

## Decision

- The repository stays one module; keel is imported only by `sdk/adapter/keel`. Core packages never import it.
- The adapter imports only keel's `port`, `common`, `model`, and `guard` packages, which pull in nothing beyond `jwt/v5` and `golang.org/x`. `outbox` and `logger` would bring cloud SDKs into every consumer, so the event side targets `port.MessagePublisher`; backing it with the outbox is a downstream choice.
- Identity is mapped from token claims (`BaseClaimIdentityMap`), never from credentials or api keys, so rotation cannot change the actor (CHR-ID-007). `Session` exposes scopes for information but nothing derives authority from them.
- `PermissionGate` layers keel RBAC behind a Charter evaluator: keel may deny an action Charter allows, never allow one Charter denies, and a capability with no mapped keel permission fails closed, mirroring an `AuthorityBinding` with `fail-closed` behavior.
- `GuardedInvoker` runs trust guards before the governed pipeline: a duplicate in flight is an unknown outcome to reconcile, a policy refusal is a denial, and neither reaches the transport or the ledger.
- `TableLogStore` keeps evidence as appended change rows keyed by namespace and id and requires a queryable `TableLogger`; on a write-only logger it fails closed rather than lose duplicate detection.
- `RuntimeContext` uses keel's request id as the execution context id and rejects one that is not a Charter identifier.

## Consequences

`go.mod` gains four modules. Round two remains open and is tracked in `todo_keel.md`: running the evaluator at keel's HTTP, CRUD, table-action, and worker boundaries; recording the tenant on evidence rows (keel's change log has no partner column); and a fixture proving namespaces and extensions cross the store unchanged. Scout composes these adapters; Charter ships no HTTP surface itself.

## Specification impact

None.
