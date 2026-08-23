# Harbor Identities and Assignments

Status: Conceptual, non-normative example

Harbor distinguishes identities from positions, assignments, credentials, sessions, and runtime instances. The Production Planning diagram in [the enterprise example](01_enterprise.md) shows position assignments visually; this page shows the different assignment forms.

| Assignment | Identity | Target kind | Target | Participation | Effective period |
|---|---|---|---|---|---|
| `ASGN-MORGAN-PP-MGR` | `HUMAN-MORGAN-LEE` | Position | `PP-MGR-01` | Occupies | 2024-01-01 onward |
| `ASGN-JORDAN-PP-PLN-01` | `HUMAN-JORDAN-KIM` | Position | `PP-PLN-01` | Occupies | 2026-01-01 onward |
| `ASGN-OEC-ORDER-EXCEPTION-SUPPORT` | `AGENT-ORDER-EXCEPTION-COORDINATOR` | Work context | `PROC-RESOLVE-ORDER-EXCEPTION` instances | Supports | 2026-04-01 through 2026-12-31 |
| `ASGN-ELENA-SALES-MANAGER` | `HUMAN-ELENA-TORRES` | Position | `POS-SALES-OPERATIONS-MANAGER` | Occupies | 2025-07-01 onward |
| `ASGN-ALEX-CREDIT-MANAGER` | `HUMAN-ALEX-RIVERA` | Position | `POS-CREDIT-MANAGER` | Occupies | 2026-01-01 onward |
| `ASGN-ALEX-CREDIT-APPROVAL` | `HUMAN-ALEX-RIVERA` | Responsibility | `RESP-APPROVE-CREDIT-EXCEPTION` | Approves | 2026-01-01 onward while `ASGN-ALEX-CREDIT-MANAGER` is active |

The work-context assignment does not place the agent in a position. The responsibility assignment does not change the organization tree. Authority is evaluated separately for each assignment context.

## Identity lifecycle examples

- `HUMAN-JORDAN-KIM` is active. A credential rotation changes the credential reference but not this identity or its historical assignments.
- Priya Shah's earlier position assignment ended on 2025-12-31. Her identity and historical evidence remain intact.
- `AGENT-ORDER-EXCEPTION-COORDINATOR` moves from proposed to active only after its definition, assignment, credentials, capability availability, and suspension controls are approved.
- Suspending the agent identity prevents new governed actions. Retiring it preserves prior evidence and requires a distinct identity for any successor.

An execution record identifies both `AGENT-ORDER-EXCEPTION-COORDINATOR` and the runtime instance that acted. A runtime replica, model, API client, session, or credential is not independently treated as the enterprise agent identity.
