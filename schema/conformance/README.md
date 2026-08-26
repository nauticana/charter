# Conformance Schemas

The `conformance_claim` schema records bounded claims for core-model, agent-runtime, and system-adapter profiles. A claim identifies the specification and profile, implementation version, exact tested rules, verification types, results, and test date. Its claim-level fields use the shared `conformanceClaimEnvelope` in `schema/common.schema.json`; `results` adds the general claim's per-rule detail.

This schema does not activate a profile or rule. Activation remains controlled by `conformance/manifest.yaml`.

`manifest`, `rule_definition`, and `fixture_descriptor` are the formats of `conformance/manifest.yaml`, the rule definitions under `conformance/rules`, and every `fixture.yaml`; the SDK's `ManifestReader` validates each file against them before use. A fixture descriptor may carry a `scenario` for runtime-behavioral rules: a base execution context and invocation, plus steps that admit work or invoke a capability with a scripted transport outcome and the decision, requirement, and evidence the implementation under test must produce.
