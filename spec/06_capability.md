# Capability Contracts

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
- **CHR-CAP-009:** A mutating capability MAY declare postconditions: externally observable effects, each with an identifier that is stable across contract versions, a statement, the declared outcomes it applies to when it does not apply to all, and whether it MUST be verified before an executed outcome may be recorded. A postcondition MAY name the declared business error its violation is recorded as.
- **CHR-CAP-010:** An action MUST NOT be recorded as executed while a verification-required postcondition applying to its outcome is violated or unknown. A violated postcondition is a failed effect unless the contract maps it to a declared business error; a failure to observe the effect after the operation was submitted is an unknown external outcome, not success. An effect observed absent MUST NOT be retried under the same idempotency key unless the contract's retry semantics declare that retry safe.
- **CHR-CAP-011:** A capability MAY type its inputs and outputs by an information shape (`CHR-INFO-010`) or an embedded JSON Schema. An invocation input MUST validate against the declared input schema before any authority check, and a failure denies the attempt. A declared outcome whose outputs do not validate against the declared output schema is an unknown external outcome. A schema MUST NOT be completed from references outside the contract.
- **CHR-CAP-012:** A precondition or postcondition MAY carry a predicate in a declared expression language, `cel` (the Common Expression Language) being the first, over the invocation input and observed facts, beside its statement. A precondition with a predicate MUST be evaluated against an observation taken for a new attempt before the external operation is submitted, and a replay of a settled idempotency key returns its recorded result without it; a violated precondition, or one whose facts cannot be observed or carry no observed-fact evidence, denies the attempt, and the action record names each such precondition with its result. A postcondition predicate states the target state an observation must show; an observer result that contradicts it leaves the postcondition unknown.

A provider's acknowledgement and the state later observed in the external system are different facts. A postcondition names the second; how it is observed belongs to the binding, not to the contract.

Transport bindings such as MCP, REST, events, vendor APIs, or in-process providers are downstream realizations of the same business contract.
