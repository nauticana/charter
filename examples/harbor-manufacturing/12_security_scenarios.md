# Security Scenarios

Status: Conceptual, non-normative example

| Scenario | Expected behavior and evidence |
|---|---|
| Suspended agent identity | Reject new work within declared enforcement latency; preserve prior and in-flight reconciliation evidence |
| Revoked credential | Authentication fails without changing the stable identity; no governed action begins |
| Embedded malicious instruction | Treat attachment text as untrusted; it cannot override policy or grant authority |
| Duplicate event | Resolve to the existing process or logical attempt; do not duplicate mutation |
| Timeout after mutation | Mark outcome unknown, reconcile by idempotency key, then retry only if safe |
| Partial stock reservation | Do not report full success; compensate, reconcile, or escalate according to contract |
| Cross-enterprise customer reference | Deny unless both actor and information access are explicitly authorized |
| External permission broader than grant | Enforce the narrower Charter authority at the action boundary |
| Evidence write unavailable | Do not initiate when required pre-action evidence cannot be preserved; after an attempted mutation, retain an unknown outcome and reconcile or escalate rather than report success |
| Capability suspended | Deny new invocations even if identity, credential, and external endpoint remain active |

Credentials and secrets are isolated from prompts, logs, ordinary evidence, and outputs. Authorization is enforced by a trusted action-boundary component rather than accepted from the agent's self-report. Security-relevant failures remain visible and protected evidence is access-controlled and alteration-evident to the declared assurance level.

These are behavior examples, not a security certification or substitute for deployment-specific threat analysis.
