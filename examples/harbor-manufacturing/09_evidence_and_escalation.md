# Evidence, Exceptions, and Escalation

Status: Conceptual, non-normative example

`EVID-OE-2026-0042` is a bundle of linked records, not an undifferentiated transcript.

| Evidence category | Record | Example |
|---|---|---|
| Observed fact | `EVR-0042-STOCK-OBSERVED` | SAP S/4HANA Cloud reported 40 eligible units at `2026-06-18T17:03:12Z` |
| Agent assertion | `EVR-0042-CLASSIFICATION` | The exception was classified as material-and-credit |
| Human decision | `EVR-0042-CREDIT-DECISION` | Credit Manager approved exposure up to USD 18,500 until the recorded expiry |
| External response | `EVR-0042-SAP-RESERVATION` | SAP S/4HANA Cloud returned reservation `0000088421` during reconciliation |
| Derived conclusion | `EVR-0042-COVERAGE` | The requested 30 units are covered after subtracting 8 existing allocations from the 40 observed |

Every governed action record links the actor identity, runtime and execution context, assignment or responsibility, capability, time, authority and approval evaluations, binding, material inputs, and outcome. Source references and transformations make provenance verifiable to the declared level.

When a late goods issue reduced availability to 38 units, `EVR-0042-STOCK-CORRECTED` superseded `EVR-0042-STOCK-OBSERVED` and explained the correction. The original remains addressable; it is not silently edited. `ACT-0042-EXECUTE-1` links the action to its authority and approval evaluations and to these records.

## Exception and escalation example

`EXC-OE-2026-0042-01` states that the stock-reservation response was not received within the expected time, affects `TASKINST-OE-0042-EXECUTE`, names the Sales Operations Manager as escalation target, and moved through disposition `reconciling` to `resolved`. Its escalation `ESC-OE-2026-0042-01` records the ambiguous outcome, urgency, requested decision, attempted operation, idempotency key, and known evidence. Reconciliation links the SAP response and closes the disposition without deleting the exception history.

Retention and access follow the underlying order, customer, approval, and security classifications. A sufficient business rationale is retained; secrets and sensitive private reasoning traces are not required.
