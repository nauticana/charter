# Keel Adapter

Optional bridge from Charter's neutral contracts to the primitives keel exposes; Charter's core packages never import keel, and keel never imports Charter. Only keel's `port`, `common`, `model`, and `guard` packages are used, so no cloud SDK enters the build.

- `SessionFromContext` reads the principal, subject, tenant, api key, scopes, and request id keel's middlewares bind; `RuntimeContext` turns the request id into the execution context of an action.
- `IdentityMap` connects keel subjects and user ids to Charter identities (`BaseClaimIdentityMap` reads them from token claims); `Caller` resolves the acting identity active at the action time.
- `PermissionGate` is an `authority.Evaluator` that layers keel's `CheckActionPermission` behind a Charter evaluator: keel may deny, never widen, and unmapped capabilities fail closed.
- `GuardedInvoker` runs a `guard.TrustGuard` chain before the governed pipeline; duplicates report an unknown outcome to reconcile, policy refusals deny.
- `TableLogStore` is an `evidence.Store` over a queryable `port.TableLogger`; `PublishingSink` publishes every appended document through `port.MessagePublisher` (back it with the outbox downstream).
- `BigintIDs` mints evidence ids from `port.BigintGenerator`; `MetricsInvoker` counts invocations through `port.MetricsRecorder`.
