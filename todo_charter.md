# Charter interfaces and reusable foundations

Charter owns the functional contracts used by downstream applications. It provides interfaces plus optional `Abstract*` extension points and complete `Base*` implementations where a neutral reusable default is possible. The specification, schemas, conformance corpus, and Go SDK require no platform; optional adapters may connect Charter to Keel, Scout, or another implementation.

## SDK contracts

- [x] Generate or maintain typed SDK models for every schema in `schema/catalog.json`, including document envelopes, references, versions, and namespaced extensions (`sdk/model`, hand-maintained per schema domain and round-trip tested against every Harbor document and its schema).
- [x] Define provider/resolver interfaces for organization, process, architecture, identity, authority, capability, binding, evidence, and information-governance documents. Organization interfaces are neutral and do not implement organization management.
- [x] Define authorization, approval, capability invocation, policy, evidence, and conformance request/decision interfaces with fail-closed results and stable requirement identifiers (`authority.Evaluator`/`ApprovalGate`, `capability.Invoker`, `information.Evaluator`, `evidence.Sink`, `validate.Runner.Claim`).
- [x] Keep SDK contracts platform-neutral and provide optional adapters for Keel's principal, evaluator, invocation, event, and validation ports. Keel never depends on Charter, and Charter never depends on Scout. (`sdk/adapter/keel` against keel v1.2.55: `Session`/`IdentityMap`/`Caller` for the principal, `PermissionGate` over `CheckActionPermission` as the evaluator, `GuardedInvoker` over trust guards for invocation, `TableLogStore`/`PublishingSink` over `port.TableLogger`/`port.MessagePublisher` for evidence events, `BigintIDs`, `MetricsInvoker`. Validation stays in `sdk/validate`; keel has no validator port, as `todo_keel.md` records.)

## Reusable foundations

- [x] Provide `Abstract*` extension points with compile-time interface assertions: `AbstractEvaluator` and `AbstractApprovalGate` (authority), `AbstractInvoker` (capability), `AbstractSink` (evidence), `AbstractDocumentProvider` (corpus), `AbstractBinding` (binding), `AbstractResolver` (identity), with `Base*` defaults where a complete neutral implementation exists.
- [x] Provide reusable reference resolution, effective-time evaluation, delegation-depth/limit evaluation, binding feature comparison, and append-only evidence helpers.
- [x] Keep persistence and external-system adapters pluggable: `corpus.Source`, `evidence.Store`, and `binding.VendorMapping`/`Executor` are the injection points; Charter, Keel, and Scout do not manage organizations.

## Conformance rules and fixtures

- [x] Publish schemas for the conformance manifest, rule definition, and fixture descriptor formats (`schema/conformance/manifest`, `rule_definition`, `fixture_descriptor`; `ManifestReader` validates every file against them).
- [x] Implement the validator interface for structural validation, semantic rule execution, fixture execution, and `ConformanceClaim` generation.
- [x] Require every active rule to parse, cite normative requirements, have deterministic pass/fail semantics, and include schema-valid positive and negative fixtures with declared additional failures (`Runner.RunManifest` rejects manifests that violate this).
- [x] Run every active fixture in CI and verify exact rule outcomes; a valid fixture must pass all selected rules and an invalid fixture must fail every declared rule (`.github/workflows/ci.yml` runs gofmt, vet, and `go test -count=1 ./...`, which executes every manifest fixture through `validate.Runner`).
- [x] Activate agent-runtime and system-adapter profiles only when their executable rules and fixtures are present (agent-runtime: six runtime-behavioral rules against `validate.Subject`; system-adapter: four against `validate.AdapterSubject`; every rule has a must-carry-out and a must-refuse scenario).
