# Charter Agents

A Charter agent is an identifiable and governed digital actor that pursues assigned business outcomes within bounded authority and produces auditable results.

## Required definition

- **CHR-AGENT-001:** An agent MUST have a declared business purpose and at least one explicit responsibility or permitted coordination function.
- **CHR-AGENT-002:** An agent MUST declare its triggers, required inputs, expected outcomes, available capabilities, governing policies, and escalation conditions.
- **CHR-AGENT-003:** An agent MUST NOT expand its own responsibilities, capabilities, credentials, or authority.
- **CHR-AGENT-004:** When a required action is outside its authority, policy, capability, or applicable confidence boundary, an agent MUST stop, request approval, delegate, or escalate as configured.
- **CHR-AGENT-005:** An agent MUST expose lifecycle state sufficient to determine whether it may accept or perform work.
- **CHR-AGENT-006:** An agent action MUST be attributable to both the stable agent identity and the runtime instance or execution context that performed it.
- **CHR-AGENT-007:** An agent supporting multiple positions MUST evaluate authority in the context of the current assignment and MUST NOT automatically combine the authority of those positions.
- **CHR-AGENT-008:** A conforming agent MUST produce or reference evidence for governed decisions and actions.
- **CHR-AGENT-009:** Publishing, activating, archiving, and rolling back an agent definition version are governed actions. Each MUST be evaluated against its own authority and any separation-of-duties constraint between authoring and publishing, and its evidence MUST name the definition and the digest of the version acted on (the action's subject and material-inputs digest), the approvals evaluated, and the evaluation results relied on (`CHR-EVID-013`).
- **CHR-AGENT-010:** A definition version derived from another definition, a template, or an import from another environment MUST record the source, its version, and its digest (`CHR-CONF-015`). Recorded provenance is part of the version and MUST NOT change once the version is published.
- **CHR-AGENT-011:** An agent definition MAY declare operating limits: resource bounds such as spend per period in integer minor units with currency and exponent, rate, steps per task, or duration. A runtime or delegation MAY add operating limits; the definition's limits continue to apply, and a runtime limit MUST NOT admit a value that a definition limit of the same kind and bound direction forbids. An attempt stopped by an operating limit MUST be evidenced as denied together with the limit evaluated (`CHR-EVID-010`).

## Technology neutrality

An agent may use language models, rules, optimization, workflows, conventional software, or human input. Conformance depends on declared behavior, authority, and evidence rather than a particular implementation technology.
