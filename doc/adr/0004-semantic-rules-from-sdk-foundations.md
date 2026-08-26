# 0004: Semantic rules built on SDK foundations

Status: Accepted
Date: 2026-08-25

## Context

The core-model profile began with eight rules while the SDK implemented checks for many more requirements. CHR-CONF-006 and CHR-CONF-007 require every active rule to cite requirements and carry valid and invalid fixtures. The schemas type every reference property as `idRef` without naming the kind it must resolve to, so CHR-CONF-012 could only be enforced for `objectRef` values.

## Decision

- A conformance rule is a thin `validate.Rule` that reuses the SDK check for its requirement (`architecture.Checker`, `evidence.Queries.Lineage`, `organization.Current`, `authority.SodEvaluator`, `agent.Declares`, `process.PerformerKind`) rather than re-deriving it. Where reuse would hide later violations behind an earlier error, the rule inlines the individual checks.
- The document kinds each `idRef` property may name are declared in the schemas as `x-charter-ref-kinds`; `SchemaMeta` collects them and rule CHR-RULE-CONF-002 checks resolved references against them, with a test asserting that every `idRef` property is annotated with known kinds. An earlier Go-side map was replaced by this annotation before the first release.
- Rules skip references that do not resolve or resolve to the wrong kind; those are CHR-RULE-CONF-001 and CONF-002 findings, so findings do not cascade.
- Every rule cites its requirements identically in the YAML definition and the Go implementation; the runner rejects a manifest where they differ, where an active rule has no implementation, or where a rule's profile does not list it.
- Each rule has exactly one invalid fixture that isolates its condition; a fixture that incidentally violates another rule declares it in `also_fails`, and the runner fails a fixture that trips an undeclared rule.

## Consequences

Rule coverage tracks the SDK: the Harbor corpus is the valid fixture for every semantic rule, so a rule the SDK cannot satisfy against Harbor is a specification or example defect surfaced immediately. Adding a schema property with an `idRef` fails the coverage test until its kind is declared. Rule output is deterministic because documents are iterated in sorted order.

## Specification impact

`x-charter-ref-kinds` is a documented annotation keyword (`schema/README.md`) and the single source of the target kind for CHR-CONF-012; the rule's semantics did not change.
