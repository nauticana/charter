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
