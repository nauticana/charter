# Go Reference SDK

Status: Version 1.0.1; every package is tested against the Harbor instances and the conformance fixtures, and the optional Keel adapter targets keel v1.2.62. The supported Charter specification and schema catalogue remain at 1.0.0.

The Go reference SDK is platform-neutral. Optional adapters may connect its contracts to Keel, Scout, or another implementation, while independent implementations can conform without importing this SDK. Keel never depends on Charter, and Charter never depends on Scout.

Packages:

- `model`: schema-aligned structs for every catalogued kind, with no behavior
- `corpus`, `temporal`: document loading and indexing, the `Source` and `AbstractDocumentProvider` foundation every provider embeds; validity evaluation
- `identity`, `organization`, `agent`, `process`, `architecture`: provider and resolution contracts with corpus-backed `Base*` defaults and reusable views (lifecycle at time, coverage, admission, task context, gap consistency)
- `authority`, `capability`, `binding`: governed decision and invocation contracts: grant and approval evaluation, the `AbstractInvoker` pipeline, binding feature support and fail-closed execution
- `evidence`, `information`: append-only evidence with supersession and integrity; information-governance decisions
- `validate`: structural/semantic validation, fixture execution, and conformance claims
- `adapter/keel`: the optional bridge to keel's principal and tenant context, RBAC permission checks, trust guards, table change log, message publishing, metrics, and request ids; only this package imports keel

Conventions: interfaces, `Abstract*` embed-only extension points, and complete `Base*` implementations live in separate files; behavior is a method on a class; `model` holds structs only. Providers resolve a reference relative to the owning document's namespace (`ownerNamespace`, `ref`), exactly as the schemas define `idRef`, and every decision fails closed with a reason and the requirement it enforces.

The repository root currently owns the Go module. If the SDK later requires an independent release cadence, extraction and import-path compatibility will be planned as a versioned migration rather than assumed to be a directory-only move.
