# Order Exception Process Definition

Status: Conceptual, non-normative example

`VS-FULFILL-CUSTOMER-ORDER` realizes value by accepting, producing, and fulfilling a customer order. `PROC-RESOLVE-ORDER-EXCEPTION` is one process within that value stream. Its outcome is a blocked order that is either safely released, explicitly rejected, or escalated to an accountable authority.

```mermaid
flowchart LR
    VS["Fulfill Customer Order<br/>ValueStream"] --> Process["Resolve Order Exception<br/>BusinessProcess"]
    Process --> T1["T01 Detect and classify<br/>Task"]
    T1 --> T2["T02 Gather required facts<br/>Task"]
    T2 --> T3["T03 Prepare resolution<br/>Task"]
    T3 --> T4["T04 Obtain approval<br/>Task"]
    T4 --> T5["T05 Execute resolution<br/>Task"]
    T5 --> T6["T06 Verify and close<br/>Task"]
```

| Task | Accountable responsibility | Permitted participation |
|---|---|---|
| `TASK-OE-DETECT` | Coordinate order exception | Human or agent observes and classifies |
| `TASK-OE-GATHER` | Coordinate order exception | Human or agent observes and prepares |
| `TASK-OE-PREPARE` | Coordinate order exception | Human or agent recommends and prepares |
| `TASK-OE-APPROVE` | Applicable approval responsibility | Authorized human approves or rejects |
| `TASK-OE-EXECUTE` | Coordinate order exception | Authorized human, agent, or system executes |
| `TASK-OE-VERIFY` | Coordinate order exception | Human or agent verifies and closes or escalates |

The definition and its running instances are separate. `PROCINST-OE-2026-0042` concerns order `ORD-2026-0173`; its task instances carry execution state without changing the process definition. Dependencies, sequence, hierarchy, and responsibility assignments are modeled separately rather than inferred from this display order.

An exception record identifies the affected process and task instance, violated expectation, responsible escalation target, and required disposition. Example branches include bounded execution, approval-gated execution, separation-of-duties rejection, stale approval, and ambiguous external outcome.
