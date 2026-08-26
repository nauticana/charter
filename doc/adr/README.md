# Architecture Decision Records

Architecture decision records explain significant design choices and their consequences. They preserve rationale without placing historical discussion inside normative specification prose.

Use filenames of the form `NNNN-short-title.md`. An ADR should contain:

```markdown
# NNNN: Decision title

Status: Proposed | Accepted | Superseded | Rejected
Date: YYYY-MM-DD

## Context

## Decision

## Consequences

## Specification impact
```

An accepted ADR does not itself create a normative requirement. Any normative result must also be incorporated into `spec/`, `schema/`, and `conformance/` as applicable.

## Records

- [0001: Enterprise object-relationship model](0001-enterprise-object-relationship-model.md)
- [0002: Model and provider foundation](0002-model-and-provider-foundation.md)
- [0003: Governed invocation pipeline](0003-governed-invocation-pipeline.md)
- [0004: Semantic rules built on SDK foundations](0004-semantic-rules-from-sdk-foundations.md)
- [0005: Runtime-behavioral conformance harness](0005-behavioral-conformance-harness.md)
- [0006: Keel adapter boundary](0006-keel-adapter-boundary.md)
