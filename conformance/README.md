# Charter Conformance

Status: Framework defined; no active conformance profile has been released.

Conformance combines normative requirement traceability, structural schema validation, semantic rules, valid and invalid fixtures, and runtime behavioral checks where static documents are insufficient.

`manifest.yaml` is the inventory of active and planned rules. A rule is active only when the manifest marks it active and supplies the evidence required by `CHR-CONF-006` and `CHR-CONF-007`.

## Result classes

- Structural: instance documents satisfy the applicable schemas.
- Semantic: relationships and invariants satisfy normative requirements.
- Behavioral: a running implementation demonstrates required decisions, failures, or evidence.

A conformance report must identify the specification version, profile, implementation version, rule-set revision, execution time, and individual rule outcomes.
