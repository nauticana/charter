# External-System Binding Model

Status: Version 1.0.0

Bindings connect stable Charter capability and authority semantics to a specific external product, version, or deployment without adding vendor concepts to the Charter core.

## Binding layers

- A `SystemProfile` identifies an external system, version, supported features, and operational constraints.
- A `CapabilityBinding` maps a Charter capability to external operations.
- A `DataBinding` maps Charter information to external representations.
- An `AuthorityBinding` maps Charter authority to external permission enforcement.
- An `EventBinding` maps business events and delivery semantics.
- An adapter is executable downstream software that realizes one or more bindings.

## Requirements

- **CHR-BIND-001:** Every binding MUST identify its Charter specification version, target system profile, and binding version.
- **CHR-BIND-002:** A binding MUST declare which capability-contract features it supports, partially supports, or does not support.
- **CHR-BIND-003:** Vendor-specific properties MUST use a namespace that cannot collide with Charter-defined properties.
- **CHR-BIND-004:** A binding that defines or carries extensions MUST satisfy the extension requirements in `CHR-CONF-002`.
- **CHR-BIND-005:** A capability binding MUST map inputs, outputs, business errors, authority checks, idempotency, and evidence obligations.
- **CHR-BIND-006:** An authority binding MUST fail closed when it cannot preserve a required Charter authorization constraint.
- **CHR-BIND-007:** Lossy data mappings MUST be declared and MUST identify their effect on validation, decisions, and evidence.
- **CHR-BIND-008:** Adapter authentication, retry, transaction, and monitoring behavior MUST be documented without becoming normative vendor behavior in the Charter core.
- **CHR-BIND-009:** Binding conformance MUST be evaluated against declared features rather than inferred from successful connectivity.

Vendor profiles, mappings, adapters, and proprietary accelerator content belong downstream unless adopted as non-normative examples.
