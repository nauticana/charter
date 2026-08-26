# 0003: Governed invocation pipeline

Status: Accepted
Date: 2026-08-25

## Context

A runtime such as Scout needs one composition that evaluates every action against authority, approval, separation of duties, information governance, binding support, and idempotency, executes through a vendor-specific transport, verifies the outcome against the contract, records evidence, and escalates stopped actions when configured. The specification requires each gate to fail closed (CHR-AUTH-010), every governed decision to be evidenced (CHR-AGENT-008), replay and partial-failure risks to be addressed (CHR-SEC-008), and outcomes and errors to stay within the contract (CHR-CAP-001, CHR-CAP-005, CHR-CAP-007).

## Decision

`capability.AbstractInvoker` runs a fixed gate order around an abstract `binding.Executor`:

1. Contract resolution and version compatibility (`VersionPolicy`; same major, offered minor not older).
2. Actor lifecycle: the actor must be active at the action time (CHR-ID-005), independently of any admission the runtime performed earlier.
3. Authority through `authority.Evaluator`. A read-class contract may proceed on its assignment alone only when `ReadWithoutGrant` is set explicitly.
4. Approval through `authority.ApprovalGate` when the contract requires it; the approval binds the approved action (which may differ from the capability, hence `Invocation.ApprovedAction`), the material-inputs digest, the subjects, and its limits, and must be valid at the action time.
5. Separation of duties over the actor's recorded, performed actions, scoped by action subjects (`authority.SubjectScope`, `authority.Performed`).
6. Information governance for every declared information use and purpose; each evaluation is recorded and any result other than allowed denies the action.
7. Binding resolution and declared feature support; a binding is required even for in-process realizations, because every realization is a binding.
8. Idempotency for mutating contracts: a key is required, a completed key replays the prior result without execution, an in-flight or unknown key blocks retry until reconciled through the ledger.
9. Transport. An error wrapping `binding.ErrNotExecuted` proves nothing happened and releases the key; any other error is an unknown outcome and marks the key unknown.
10. Verification: an outcome or business error the contract does not declare is reported as unknown, never as success.
11. Evidence: an `ActionRecord` is appended for every non-replay pipeline decision, including denials. A completed replay returns its previously recorded result without appending another action. A gate that denies before authority was evaluated records the authority evaluation as `error: not evaluated`; an approval that was required but never evaluated is recorded as `missing`. No record is written for pre-contract composition or invocation-shape failures, or when the contract itself cannot be resolved, because no valid operation class is available.
12. Escalation: after evidence is recorded, a composed `Escalator` may append an exception and escalation for configured stopped dispositions; escalation failures are returned rather than suppressed.

Approval evaluation lives in `authority`, not `capability`, because approvals are authority semantics; the separation-of-duties primitives live there for the same reason.

## Consequences

A runtime composes the invoker with its own sources, ledger, sink, and transport; platform adapters decorate it (trust guards, metrics) rather than reimplementing gates. Reconciliation after an unknown outcome is the adapter's responsibility, completed through `Ledger.Complete`. Reasons and outcomes are clipped to the schema's 500-character limit. The behavioral harness (ADR 0005) found that the lifecycle gate was missing from the first version; it is now step 2.

## Specification impact

CHR-EVID-010 now requires a denied, failed, or unknown attempt to be evidenced like an executed action with an explicit disposition and the evaluations made before it stopped; `action_record.schema.json` carries the required `disposition` and optional `informationEvaluations`, and CHR-RULE-CAP-001 checks recorded outcomes against the contract. Step 11 above implements that requirement.
