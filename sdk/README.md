# Go Reference SDK

Status: Draft; `model`, `corpus`, `temporal`, `validate`, and `authority` are implemented and tested against the Harbor instances and the conformance fixtures; the remaining packages are planned.

The Go reference SDK is platform-neutral. Optional adapters may connect its contracts to Keel, Scout, or another implementation, while independent implementations can conform without importing this SDK. Keel never depends on Charter, and Charter never depends on Scout.

Planned packages:

- `model`: schema-aligned structs with no behavior
- `corpus`, `temporal`: document loading and indexing; validity evaluation
- `identity`, `organization`, `agent`, `process`, `architecture`: provider and resolution contracts
- `authority`, `capability`, `binding`: governed decision and invocation contracts
- `evidence`, `information`: evidence and information-governance contracts
- `validate`: structural/semantic validation, fixture execution, and conformance claims

Conventions: interfaces, `Abstract*` embed-only extension points, and complete `Base*` implementations live in separate files; behavior is a method on a class; `model` holds structs only.

The repository root currently owns the Go module. If the SDK later requires an independent release cadence, extraction and import-path compatibility will be planned as a versioned migration rather than assumed to be a directory-only move.
