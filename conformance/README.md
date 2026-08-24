# Charter Conformance

Status: core-model profile active with eight semantic rules; agent-runtime and system-adapter profiles planned.

Conformance combines normative requirement traceability, structural schema validation, semantic rules, valid and invalid fixtures, and runtime behavioral checks where static documents are insufficient.

`manifest.yaml` is the inventory of profiles and rules. Each active rule has a definition in `rules/` citing its `CHR-*` requirements, a valid fixture (the Harbor instance documents), and an invalid fixture under `fixtures/invalid/` that is schema-valid and fails that rule.

## Result classes

- Structural: instance documents satisfy the applicable schemas.
- Semantic: relationships and invariants satisfy normative requirements.
- Behavioral: a running implementation demonstrates required decisions, failures, or evidence.

A conformance report identifies the specification version, profile, implementation version, rule-set revision, execution time, and individual rule outcomes.
