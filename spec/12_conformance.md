# Conformance and Compatibility

Status: Version 1.1.0

Conformance allows independent implementations to make bounded, testable claims about the parts of Charter they support.

## Profiles

The profiles are:

- Core model: structural and semantic support for Charter documents
- Agent runtime: identity, lifecycle, authority evaluation, escalation, and evidence
- System adapter: system profiles, bindings, capability execution, and error preservation

Profiles are not active until their required rules are listed in `conformance/manifest.yaml`.

## Requirements

- **CHR-CONF-001:** A conforming artifact or implementation MUST identify the Charter specification version and conformance profile it targets.
- **CHR-CONF-002:** Extensions MUST use a namespace that cannot collide with Charter-defined properties and MUST NOT change or cause reinterpretation of Charter-defined semantics.
- **CHR-CONF-003:** An implementation MUST reject input that violates a required structural or semantic constraint rather than silently coercing it into conformance.
- **CHR-CONF-004:** Identifiers intended to persist across integrations or audit records MUST be stable within their declared namespace.
- **CHR-CONF-005:** A conformance claim MUST identify the specification version, profile, implementation version, tested rule set, and result date.
- **CHR-CONF-006:** Every active conformance rule MUST reference at least one normative requirement identifier.
- **CHR-CONF-007:** Every machine-testable active rule MUST have at least one valid fixture and one invalid fixture unless the manifest documents why one class is inapplicable.
- **CHR-CONF-008:** A conforming validator MUST produce an identifiable failure for an invalid fixture and MUST NOT report it as conforming.
- **CHR-CONF-009:** Partial support MUST be reported as partial and MUST identify omitted or unsupported features.
- **CHR-CONF-010:** A minor or patch release MUST NOT intentionally invalidate an artifact that conformed to the preceding release in the same major version.
- **CHR-CONF-011:** Conformance results MUST distinguish structural validation, semantic validation, and runtime behavioral verification.
- **CHR-CONF-012:** A reference to a Charter document MUST resolve within its declared namespace to a document of the required kind. A reference to a kind outside the Charter catalogue MUST be explicitly declared external; only such declared external references are exempt from Charter document resolution.
- **CHR-CONF-013:** A runtime-behavioral rule MUST be defined by executable scenarios in which the verification harness scripts every external outcome and the implementation under test makes every governed decision. Its valid fixture is a scenario a conforming implementation MUST carry out and its invalid fixture one it MUST refuse, each stating the decision, the requirement identifier, and the evidence the implementation must produce.
- **CHR-CONF-014:** A conformance claim covering runtime-behavioral rules MUST identify the implementation whose behavior was exercised. A rule whose implementation was not exercised MUST be reported as not tested, and a claim containing such a rule MUST be reported as partial.

## Compatibility

The specification, schemas, and conformance manifest release together. Examples and SDK releases declare the specification versions they support. Breaking normative changes require a major version and migration guidance.
