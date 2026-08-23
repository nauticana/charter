# Capability Contracts

Status: Draft, unreleased

A capability contract describes the stable business meaning of an operation independently of the protocol or vendor system that implements it.

## Requirements

- **CHR-CAP-001:** A capability MUST declare a stable identifier, business purpose, operation class, inputs, outputs, preconditions, and possible outcomes.
- **CHR-CAP-002:** The operation class MUST distinguish read, propose, approve, and execute behavior where applicable.
- **CHR-CAP-003:** A capability MUST declare required authority and any approval, separation-of-duties, or transaction-limit constraints.
- **CHR-CAP-004:** Mutating capabilities MUST declare idempotency and retry semantics.
- **CHR-CAP-005:** A capability MUST define business errors independently of vendor transport or protocol errors.
- **CHR-CAP-006:** A capability contract MUST identify the evidence required for invocation, decision, execution, and outcome.
- **CHR-CAP-007:** Implementations MUST NOT broaden the business meaning or effective authority of a capability when mapping it to an external operation.
- **CHR-CAP-008:** A capability version change that invalidates previously valid inputs, outputs, authority, or outcomes MUST be treated as breaking.

Transport bindings such as MCP, REST, events, vendor APIs, or in-process providers are downstream realizations of the same business contract.
