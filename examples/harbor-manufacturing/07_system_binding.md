# Harbor Operations System Binding

Status: Conceptual, non-normative example

Harbor Operations System (HOS) is fictional. `SYSPROFILE-HOS-2026-01` identifies its fictional API version, supported features, authentication method, transaction boundaries, delivery behavior, limits, and operational constraints. `BIND-HOS-ORDER-EXCEPTION-1` targets that profile and declares its Charter specification and binding versions.

## Binding kinds

| Kind | Example mapping |
|---|---|
| CapabilityBinding | `CAP-RESERVE-ORDER-STOCK` to `POST /stock-reservations` |
| DataBinding | Charter order, line, quantity, money, and expiry meanings to HOS request and response fields |
| AuthorityBinding | Charter grant and approval evaluation to the enforcement context accepted by HOS |
| EventBinding | HOS `order.blocked` delivery to the declared `order-blocked` trigger |
| BindingConformance | Supported, partial, or unsupported result for each declared feature |

The capability binding maps inputs, outputs, business and transport errors, authority checks, idempotency, retry, and evidence obligations. Authentication and monitoring are documented as adapter behavior, not Charter core semantics.

HOS represents reservation expiry to the minute. If a Charter input carries finer precision, the DataBinding declares the loss, its rounding rule, and its effects on validation and evidence. A mapping that cannot preserve an authority limit fails closed.

The fictional extension `https://harbor.example/ns/warehouse#allocationStrategy` cannot collide with Charter properties or reinterpret them. The adapter is downstream executable software; successful connectivity alone is not binding conformance.
