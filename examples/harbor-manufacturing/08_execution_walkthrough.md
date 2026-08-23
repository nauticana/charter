# Order Exception Execution Walkthrough

Status: Conceptual, non-normative example

This walkthrough connects the example objects through `PROCINST-OE-2026-0042`, concerning a material and credit hold on order `ORD-2026-0173`.

```mermaid
sequenceDiagram
    participant SAP as SAP S/4HANA Cloud
    participant Agent as Order Exception Coordinator
    participant Credit as Credit Manager
    participant Evidence as Evidence service

    SAP->>Agent: order-blocked integration event
    Agent->>Evidence: Record trigger and execution context
    Agent->>SAP: Read required business observations
    SAP-->>Agent: Current observations
    Agent->>Evidence: Record facts, provenance, and classification
    Agent->>Credit: Submit bound credit proposal
    Credit-->>Agent: Approve with limits and expiry
    Credit->>SAP: Apply approved credit disposition
    SAP-->>Credit: Confirm hold disposition
    Agent->>Evidence: Record human decision and disposition reference
    Agent->>Agent: Recheck assignment, policy, authority, inputs, and SoD
    Agent->>SAP: Create reservation through SAP binding
    Note over SAP,Agent: Reservation request timed out and its outcome is unknown
    Agent->>SAP: Reconcile using recorded SAP references
    SAP-->>Agent: Reservation already created
    Agent->>Evidence: Record reconciliation and verified outcome
    Agent->>SAP: Read final order status
    Agent->>Evidence: Close or escalate process instance
```

## Trace

| Step | Task instance | Performer mode | Principal artifacts |
|---|---|---|---|
| 1 | `TASKINST-OE-0042-DETECT` | Agent observes | Trigger, identity, runtime, event binding |
| 2 | `TASKINST-OE-0042-GATHER` | Agent observes | Current facts, classifications, provenance |
| 3 | `TASKINST-OE-0042-PREPARE` | Agent recommends and prepares | Candidate resolutions and proposal |
| 4 | `TASKINST-OE-0042-APPROVE` | Human approves | Bound approval, decision rationale, and applied credit disposition |
| 5 | `TASKINST-OE-0042-EXECUTE` | Agent executes | Grant evaluation, idempotency key, binding, external result |
| 6 | `TASKINST-OE-0042-VERIFY` | Agent observes | Reconciliation, verified outcome, closure evidence |

The timeout does not cause an immediate second mutation. The agent treats the outcome as unknown, reconciles it through `BIND-S4-RESERVE-ORDER-STOCK-1`, discovers the existing SAP reservation, and records `already-reserved`. The process definition remains unchanged by these instance states.
