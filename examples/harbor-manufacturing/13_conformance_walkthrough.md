# Conformance Walkthrough

Status: Illustrative only; not a conformance claim

The core-model profile is active with twenty semantic rules, the agent-runtime profile with eight runtime-behavioral rules, and the system-adapter profile with four; the Harbor instance documents are the valid fixture for the former and the document set most behavioral scenarios drive. This page shows what a result for the Harbor example would contain without making a conformance claim.

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

## Fixture coverage

The Harbor documents are the valid fixture for every semantic rule and the document set behind most behavioral scenarios: the stable object graph, complete agent definition, scoped grant, bound approval, idempotent reservation, evidence linkage, and declared binding support are all exercised. Invalid fixtures independently demonstrate a cyclic organization tree, a position used as an organization-tree node, an inactive identity acting, authority over limit, a self-approval conflict, a stale approval, and unsafe duplicate execution. An undeclared extension and a missing evidence attribution are rejected by the schemas themselves, so no semantic rule is needed; a silent lossy mapping is not machine-checkable from documents and remains a review concern.

Each rule references one or more normative `CHR-*` requirement identifiers and has valid and invalid fixtures. Partial binding support is reported as partial. A validator rejects rather than silently repairs invalid input.

The reference SDK's claims for release 1.0.0 are published under `conformance/claims/`; the Harbor example itself makes no claim.
