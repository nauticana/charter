# Evidence, Exceptions, and Escalation

Status: Conceptual, non-normative example

`EVID-OE-2026-0042` is a bundle of linked records, not an undifferentiated transcript.

| Evidence category | Example record |
|---|---|
| Observed fact | SAP S/4HANA Cloud reported 40 eligible units at `2026-06-18T17:03:12Z` |
| Agent assertion | The exception was classified as material-and-credit |
| Human decision | Credit Manager approved the stated exposure until the recorded expiry |
| External response | SAP S/4HANA Cloud returned the example reservation identifier `RSV-88421` during reconciliation |
| Derived conclusion | The requested quantity is covered after subtracting existing allocations |

Every governed action record links the actor identity, runtime and execution context, assignment or responsibility, capability, time, authority and approval evaluations, binding, material inputs, and outcome. Source references and transformations make provenance verifiable to the declared level.

If the observed available quantity is later corrected, a new evidence record supersedes the old observation and explains the correction. The original remains addressable; it is not silently edited.

## Exception and escalation example

`EXC-OE-2026-0042-01` states that the stock-reservation response was not received within the expected time, affects `TASKINST-OE-0042-EXECUTE`, has disposition `reconciling`, and names the Sales Operations Manager as escalation target. Its escalation records the ambiguous outcome, urgency, requested decision, attempted operation, idempotency key, and known evidence. Reconciliation later links the SAP response and changes the disposition to `resolved` without deleting the exception history.

Retention and access follow the underlying order, customer, approval, and security classifications. A sufficient business rationale is retained; secrets and sensitive private reasoning traces are not required.
