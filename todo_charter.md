# Charter interfaces and reusable foundations

Charter owns the functional contracts used by downstream applications. It provides interfaces plus optional `Abstract*` extension points and complete `Base*` implementations where a neutral reusable default is possible. The specification, schemas, conformance corpus, and Go SDK require no platform; optional adapters may connect Charter to Keel, Scout, or another implementation.

## SDK contracts

- [ ] Generate or maintain typed SDK models for every schema in `schema/catalog.json`, including document envelopes, references, versions, and namespaced extensions.
- [ ] Define provider/resolver interfaces for organization, process, architecture, identity, authority, capability, binding, evidence, and information-governance documents. Organization interfaces are neutral and do not implement organization management.
- [ ] Define authorization, approval, capability invocation, policy, evidence, and conformance request/decision interfaces with fail-closed results and stable requirement identifiers.
- [ ] Keep SDK contracts platform-neutral and provide optional adapters for Keel's principal, evaluator, invocation, event, and validation ports. Keel never depends on Charter, and Charter never depends on Scout.

## Reusable foundations

- [ ] Provide `Abstract*` extension points with compile-time interface assertions: `AbstractAuthorityEvaluator` (lookup abstract; validity, limits, delegation depth, SoD, structured decision done), `AbstractCapabilityInvoker` (transport and error mapping abstract; authority, approval, binding, verification, evidence pipeline done), `AbstractEvidenceSink` (storage abstract; append-only, supersession, integrity done), `AbstractDocumentProvider` (loading abstract; validation, resolution, effective-time queries done), `AbstractBinding` (vendor mapping abstract; feature support and fail-closed handling done), `AbstractApprovalGate` (routing abstract; digest, staleness, validity mode done). `Base*` defaults where a complete neutral implementation exists.
- [ ] Provide reusable reference resolution, effective-time evaluation, delegation-depth/limit evaluation, binding feature comparison, and append-only evidence helpers.
- [ ] Keep persistence and external-system adapters pluggable. Applications or optional bindings supply organization documents through Charter's neutral provider and resolution contracts; Charter, Keel, and Scout do not manage organizations.

## Conformance rules and fixtures

- [ ] Publish schemas for the conformance manifest, rule definition, and fixture descriptor formats.
- [ ] Implement the validator interface for structural validation, semantic rule execution, fixture execution, and `ConformanceClaim` generation.
- [ ] Require every active rule to parse, cite normative requirements, have deterministic pass/fail semantics, and include schema-valid positive and negative fixtures with declared additional failures.
- [ ] Run every active fixture in CI and verify exact rule outcomes; a valid fixture must pass all selected rules and an invalid fixture must fail every declared rule.
- [ ] Activate agent-runtime and system-adapter profiles only when their executable rules and fixtures are present.
