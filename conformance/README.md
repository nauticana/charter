# Charter Conformance

Status: core-model profile active with twenty semantic rules; agent-runtime profile active with eight runtime-behavioral rules and system-adapter profile with four, both run against the reference SDK by default.

Conformance combines normative requirement traceability, structural schema validation, semantic rules, valid and invalid fixtures, and runtime behavioral checks where static documents are insufficient.

`manifest.yaml` is the inventory of profiles and rules. Each active rule has a definition in `rules/` citing its `CHR-*` requirements, a valid fixture (the Harbor instance documents), and an invalid fixture under `fixtures/invalid/` that is schema-valid and fails that rule.

## Result classes

- Structural: instance documents satisfy the applicable schemas.
- Semantic: relationships and invariants satisfy normative requirements.
- Behavioral: a running implementation demonstrates required decisions, failures, or evidence. A behavioral fixture carries a `scenario` (see `schema/conformance/fixture_descriptor.schema.json`) that the harness drives against a subject under test with a scripted transport; `expected: pass` scenarios must be carried out and `expected: fail` scenarios must be refused, each with the status, requirement id, and evidence the scenario states. The SDK is the default subject; another runtime implements `validate.Subject`, and another adapter `validate.AdapterSubject`, whose request, response, and error mapping the harness judges over a scripted vendor endpoint.

`manifest.yaml`, every rule definition, and every `fixture.yaml` are validated against the format schemas in `schema/conformance/` before use.

A conformance report identifies the specification version, profile, implementation version, rule-set revision, execution time, and individual rule outcomes.
