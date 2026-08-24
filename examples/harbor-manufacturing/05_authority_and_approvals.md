# Authority and Approval Examples

Status: Conceptual, non-normative example

Authority is independent of identity, position membership, role, assignment, and external-system permission. Each governed execution evaluates an explicit grant in its current context.

## Grant and delegation

`AUTH-OEC-STOCK-RESERVATION-2026` records the Sales Operations Manager as delegator, `AGENT-ORDER-EXCEPTION-COORDINATOR` as recipient, `CAP-RESERVE-ORDER-STOCK` as operation, assigned Harbor order exceptions as resource scope, USD 25,000 equivalent and 72 hours as limits, the 2026 validity period, and a redelegation depth of 0.

Revocation or expiry prevents new reservations but does not invalidate evidence for reservations performed while the grant was effective.

## Bound approval

`APPR-OE-2026-0042-CREDIT-01` binds:

- The Credit Manager identity acting under `RESP-APPROVE-CREDIT-EXCEPTION`
- Process instance `PROCINST-OE-2026-0042`
- The proposed credit disposition and customer/order identifiers
- Material values, currency, credit observation, and proposal digest
- Maximum exposure of USD 18,500 and the condition that order value and credit observation are unchanged
- Issued 2026-06-18T16:00:00Z, expiring 2026-06-19T16:00:00Z

Changing a material input makes this approval stale. Approval is a decision record; it does not execute a release.

## Evaluation cases

| Case | Expected result |
|---|---|
| Active grant, matching scope, USD 18,500, 48 hours | Reservation may proceed after other checks |
| No grant can be established | Deny |
| Grant expired one minute before execution | Deny and escalate |
| USD 27,000 reservation | Deny or request separately authorized action |
| Agent prepared and attempts to approve credit proposal | Reject `SOD-PREPARE-APPROVE-CREDIT` conflict |
| Approval exists but order value changed | Reject stale approval |
| External permission exists but Charter grant does not | Deny |

Denials identify the missing, exceeded, expired, or conflicting authority rather than appearing as transport failures.
