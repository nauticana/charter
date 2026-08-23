# Conformance Walkthrough

Status: Illustrative only; not a conformance claim

Charter profiles are inactive until the conformance manifest activates their rules. This page shows what a future result for the Harbor example would contain without claiming that the current Markdown conforms.

## Illustrative claim envelope

| Property | Example value |
|---|---|
| Artifact | Harbor order-exception example bundle |
| Charter specification version | Future released version, not yet assigned |
| Profile | Core model, Agent runtime, or System adapter as applicable |
| Implementation version | Version of the tested validator, runtime, or adapter |
| Tested rules | Exact active `CHR-RULE-<AREA>-<NUMBER>` identifiers |
| Result date | Test execution time |
| Result classes | Structural, semantic, and runtime behavioral |

## Future fixture coverage

Valid fixtures should exercise the stable object graph, complete agent definition, scoped grant, bound approval, idempotent reservation, evidence linkage, and declared binding support. Invalid fixtures should independently demonstrate a cyclic organization tree, position used as an organization-tree node, inactive identity action, authority over limit, self-approval conflict, stale approval, undeclared extension, silent lossy mapping, missing evidence attribution, and unsafe duplicate execution.

Each rule references one or more normative `CHR-*` requirement identifiers and has valid and invalid fixtures unless the manifest documents why one class does not apply. Partial binding support is reported as partial. A validator rejects rather than silently repairs invalid input.

Until schemas and rules are released, this documentation supplies design scenarios and traceability targets only.
