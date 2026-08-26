# Authority, Delegation, and Approval

Status: Version 1.0.0

Authority determines which actor may perform which operation, on which resources, under which conditions and limits.

## Requirements

- **CHR-AUTH-001:** Authority MUST be explicitly scoped to an actor, operation or capability, resource scope, organizational context, and validity period.
- **CHR-AUTH-002:** Absence of a grant MUST be treated as denial for governed execution.
- **CHR-AUTH-003:** Delegation MUST identify the delegating authority, recipient, delegated scope, validity period, and the permitted redelegation depth.
- **CHR-AUTH-004:** A delegate MUST NOT exercise or redelegate authority beyond the delegator's effective authority.
- **CHR-AUTH-005:** Revocation or expiry MUST prevent new actions and MUST NOT alter evidence for actions completed while authority was valid.
- **CHR-AUTH-006:** Proposal, approval, and execution MUST be distinct operations even when one actor is permitted to perform more than one of them.
- **CHR-AUTH-007:** Separation-of-duties constraints MUST be evaluated before execution and MUST identify the conflicting assignments or actions.
- **CHR-AUTH-008:** Transaction, value, data, and resource limits MUST be evaluated using the current action context.
- **CHR-AUTH-009:** Approval MUST bind the approver, approved action or action class, material inputs, limits, and validity conditions.
- **CHR-AUTH-010:** An implementation MUST fail closed when required authority cannot be established.

External authorization systems may enforce these requirements, but their native roles or permissions do not replace the Charter authority semantics.

Monetary limits use integer minor units and declare the currency exponent. This preserves exact comparisons and supports currencies whose exponent is not two.
