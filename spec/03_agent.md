# Charter Agents

Status: Version 1.0.0

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

## Technology neutrality

An agent may use language models, rules, optimization, workflows, conventional software, or human input. Conformance depends on declared behavior, authority, and evidence rather than a particular implementation technology.
