# Capability Contract Examples

Status: Conceptual, non-normative example

The four contracts below demonstrate the operation classes without binding their business meanings to a protocol or product.

| Capability | Class | Business outcome |
|---|---|---|
| `CAP-READ-ORDER-EXCEPTION` | Read | Return the minimum current facts needed to coordinate one exception |
| `CAP-PROPOSE-ORDER-RESOLUTION` | Propose | Produce evidenced alternatives and a recommended disposition |
| `CAP-APPROVE-CREDIT-EXCEPTION` | Approve | Record an authorized credit decision bound to material inputs |
| `CAP-RESERVE-ORDER-STOCK` | Execute | Create or confirm a bounded stock reservation |

## Execute contract detail

`CAP-RESERVE-ORDER-STOCK` accepts an order, eligible lines and quantities, location scope, expiry, valuation context, authority reference, optional approval reference, and idempotency key. Preconditions include current order state, available stock, active authority, applicable limits, separation of duties, and supported binding behavior.

Its outcomes are `reserved`, `already-reserved`, `rejected`, or `unknown-pending-reconciliation`. Business errors include insufficient eligible stock, invalid order state, authority denied, limit exceeded, approval invalid, and policy conflict. Authentication failure, timeout, and malformed response are transport or integration failures and are mapped without replacing the business result.

A retry reuses the logical idempotency key. An ambiguous timeout is reconciled before another mutation. Required evidence covers invocation, input provenance, authority and approval decisions, execution reference, and verified outcome. A change that makes previously valid inputs, authority, or outcomes invalid is treated as a breaking contract change.
