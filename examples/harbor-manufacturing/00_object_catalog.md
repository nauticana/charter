# Harbor Object Catalogue

Status: Conceptual, non-normative example

This catalogue gives the recurring Harbor Manufacturing objects stable example identifiers. The identifiers are local to the fictional Harbor namespace and are not Charter-assigned identifiers.

| Kind | Identifier | Name or purpose |
|---|---|---|
| Enterprise | `ENT-HARBOR` | Harbor Manufacturing |
| OrganizationUnit | `OU-SALES-OPERATIONS` | Sales Operations |
| OrganizationUnit | `OU-PRODUCTION-PLANNING` | Production Planning |
| PositionType | `PT-FUNCTIONAL-MANAGER` | Functional manager position classification |
| Position | `POS-SALES-OPERATIONS-MANAGER` | Sales Operations Manager |
| Position | `POS-ORDER-SPECIALIST` | Order Specialist |
| Position | `POS-CREDIT-MANAGER` | Credit Manager |
| Position | `PP-MGR-01` | Production Planning Manager |
| Position | `PP-PLN-01` | Production Planner, first position |
| Role | `ROLE-ORDER-EXCEPTION-SUPPORT` | Groups work used to coordinate order exceptions |
| Responsibility | `RESP-COORDINATE-ORDER-EXCEPTION` | Bring an order exception to a governed disposition |
| Responsibility | `RESP-APPROVE-CREDIT-EXCEPTION` | Decide whether a credit exception is acceptable |
| Human identity | `HUMAN-MORGAN-LEE` | Morgan Lee |
| Human identity | `HUMAN-JORDAN-KIM` | Jordan Kim |
| Human identity | `HUMAN-ELENA-TORRES` | Elena Torres, assigned to Sales Operations Manager |
| Human identity | `HUMAN-ALEX-RIVERA` | Alex Rivera, assigned to Credit Manager |
| Agent identity | `AGENT-ORDER-EXCEPTION-COORDINATOR` | Order Exception Coordinator |
| Agent definition | `AGENTDEF-ORDER-EXCEPTION-COORDINATOR-1` | Governed design for the coordinator |
| Agent assignment | `ASGN-OEC-ORDER-EXCEPTION-SUPPORT` | Assigns the agent to exception-support work context |
| Value stream | `VS-FULFILL-CUSTOMER-ORDER` | Fulfill Customer Order |
| Business process | `PROC-RESOLVE-ORDER-EXCEPTION` | Resolve Order Exception |
| Process instance | `PROCINST-OE-2026-0042` | Example exception for order `ORD-2026-0173` |
| Capability | `CAP-READ-ORDER-EXCEPTION` | Read relevant exception facts |
| Capability | `CAP-PROPOSE-ORDER-RESOLUTION` | Prepare a resolution proposal |
| Capability | `CAP-REQUEST-EXCEPTION-APPROVAL` | Request a bound approval |
| Capability | `CAP-APPROVE-CREDIT-EXCEPTION` | Record an authorized credit decision |
| Capability | `CAP-RESERVE-ORDER-STOCK` | Reserve eligible stock |
| Authority grant | `AUTH-OEC-STOCK-RESERVATION-2026` | Bounded reservation delegation |
| Approval | `APPR-OE-2026-0042-CREDIT-01` | Credit decision for the example instance |
| System profile | `SYSPROFILE-HARBOR-S4-2602` | Harbor's example SAP S/4HANA Cloud Public Edition profile |
| Capability binding | `BIND-S4-RESERVE-ORDER-STOCK-1` | Example reservation binding to SAP |
| Evidence bundle | `EVID-OE-2026-0042` | Evidence for the example process instance |
| Baseline state | `ARCH-BASELINE-2026-Q1` | Current manually coordinated state |
| Target state | `ARCH-TARGET-2026-Q4` | Governed agent-supported state |
| Gap | `GAP-ORDER-EXCEPTION-EVIDENCE` | Missing consistent decision and execution evidence |
| Roadmap item | `ROADMAP-GOVERNED-EXCEPTION-PILOT` | Introduce governed support and measure it |

This catalogue lists the principal objects referenced across multiple Harbor documents; it is not an exhaustive registry. An individual document may define additional objects needed by its example, such as task definitions, task instances, runtime instances, execution contexts, credentials, policies, events, exceptions, escalations, or evidence records. Each additional object has its own kind and identifier, and its relationships to catalogue objects must be stated explicitly rather than inferred from identifier structure or document location.

An object keeps its identifier when only its display name changes. A materially different object receives a different identifier.
