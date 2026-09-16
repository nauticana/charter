# 0006: Keel adapter boundary

Status: Accepted
Date: 2026-08-25

## Context

Keel is the backend framework beneath the reference runtime. The adapter was originally reviewed against v1.2.55 and is currently verified against v1.2.64. Those reviews concluded that keel already exposes the primitives a Charter adapter needs (principal and tenant context keys, `CheckActionPermission`, trust guards, the table change logger, message publishing, metrics, request ids) and that everything Charter-shaped belongs in Charter or an adapter. Charter's core must stay platform-free, and keel must never depend on Charter.

## Decision

- The repository stays one module; keel is imported only by `sdk/adapter/keel`. Core packages never import it.
- The adapter imports only keel's `port`, `common`, `model`, and `guard` packages, which pull in nothing beyond `jwt/v5` and `golang.org/x`. `outbox` and `logger` would bring cloud SDKs into every consumer, so the event side targets `port.MessagePublisher`; backing it with the outbox is a downstream choice.
- Identity is mapped from token claims (`BaseClaimIdentityMap`), never from credentials or api keys, so rotation cannot change the actor (CHR-ID-007). The map returns the corresponding keel grant principal: a human uses its claimed keel user id, while an agent uses `AgentPrincipalKind`, its exact stable Charter `AgentIdentity` id, and the authenticated `CallerSession.PartnerID` as its sole `tenant_id` scope. This is the contract for a downstream's agent grant source. `common.CallerSession` exposes token scopes for information, but they do not grant Charter authority.
- `PermissionGate` passes that principal to keel's principal-aware RBAC behind a Charter evaluator: keel may deny an action Charter allows, never allow one Charter denies, and a capability with no mapped keel permission fails closed, mirroring an `AuthorityBinding` with `fail-closed` behavior. A compile-time interface assertion keeps its `PermissionChecker` signature aligned with `port.DatabaseRepository`.
- `GuardedInvoker` runs trust guards before the governed pipeline: a duplicate in flight is an unknown outcome to reconcile, a policy refusal is a denial, and neither reaches the transport or the ledger.
- `TableLogStore` keeps evidence as appended change rows keyed by namespace and id and requires a `ChangeLog`, a `port.TableLogger` that is also a `port.ChangeQuerier`; a write-only logger is refused at composition rather than lose duplicate detection.
- `Ledger` backs `capability.Ledger` with `port.IdempotencyLedger`, storing completed results as JSON; the fenced claim keel grants reaches the caller as `Result.LedgerFence` so an unknown outcome can be reconciled.
- `RuntimeContext` uses keel's request id as the execution context id and rejects one that is not a Charter identifier.
- `Gate` protects keel's HTTP, table-action, and worker boundaries without importing keel's `handler` or `worker` packages, which would pull the cloud SDKs in: a middleware over a route-to-capability table, a handler wrapper that sits where `handler.WrapTableAction` does, and `Check` under `common.WithCallerSession` for workers. A gate establishes the active identity and an effective grant only; limits, approvals, separation of duties, and information governance remain at the action boundary in the invoker (CHR-SEC-003).

## Consequences

`go.mod` gains four modules. Nothing is requested from keel: tenant scoping on change rows arrived with keel v1.2.55. Compatibility with v1.2.62 moved the OAuth token identity from `port.Principal` to `model.TokenPrincipal` and changed `PermissionChecker` plus `IdentityMap` to pass the separate grant-evaluation `model.Principal`, so agent actors use their own tenant-scoped keel grant source without a synthetic user id; the grant row's `agent_id` is the stable Charter identity id. keel v1.2.64 upstreamed the caller session (`common.CallerSession`), the fenced idempotency ledger (`port.IdempotencyLedger`), and the query side of the table logger (`port.ChangeQuerier`); the adapter consumes them instead of carrying its own. One generic piece stays an upstream candidate: a pre-action port inside keel's generated CRUD. Scout composes these adapters; Charter ships no HTTP surface itself.

## Specification impact

None.
