# Scout interfaces required by Charter

Scout is the agent runtime and MCP server; it owns Charter agent identity, definition, and runtime behavior. It uses Charter SDK functional contracts and adapts them to Keel's domain-independent backend ports. Organization documents come from the application or an optional binding through Charter interfaces; Scout does not manage them.

## Agent and work context

- [ ] Separate `AgentIdentity`, versioned `AgentDefinition`, and identifiable `AgentRuntime`; every runtime references the exact identity, definition, and version it operates (CHR-ID-003..007, CHR-AGENT-005..006).
- [ ] Align `AgentDefinition` with Charter purpose, accountability, responsibilities, triggers, inputs, outcomes, capabilities, policies, confidence boundaries, and escalation conditions; Scout-specific metadata uses namespaced extensions (CHR-AGENT-001..004).
- [ ] Expose effective-time lifecycle queries for identities, definitions, and runtimes. `draining` remains a runtime condition, not an identity state (CHR-ID-004..006, CHR-SEC-009).
- [ ] Accept work only with resolved Charter task/process, assignment, participation, and organization context; evaluate one assignment context at a time and never treat an agent as a position occupant (CHR-ENT-003..007, CHR-ENT-011, CHR-AGENT-007, CHR-PROC-002..006).

## Capability and execution

- [ ] Resolve and authorize a Charter `CapabilityContract` before selecting a tool or binding; tool names, transport scopes, and host trust never grant authority (CHR-CAP-001..003, CHR-SEC-003..004).
- [ ] Use Charter system-profile and binding interfaces for every execution path, including MCP and Keel-native actions; reject bindings that cannot preserve required semantics (CHR-BIND-001..009).
- [ ] Expose one invocation/result contract preserving principal, capability/binding versions, assignment context, inputs, idempotency, declared outcomes, business errors, and transport failures (CHR-CAP-004..008).
- [ ] Apply declared event delivery, retry, and reconciliation semantics; mutating capabilities are not retried without required idempotency or reconciliation (CHR-BIND-008, CHR-SEC-008).

## Governance and evidence

- [ ] Adapt Charter authorization requests/decisions to Keel's evaluator port for every read, propose, approve, and execute path; missing or indeterminate authority fails closed (CHR-AUTH-001..010).
- [ ] Keep proposal, approval, and execution separate. Approval evaluation binds approver, action, material-input digest, limits, conditions, validity, and revocation state without dispatching execution (CHR-AUTH-006..009).
- [ ] Publish Charter `ActionRecord` and `EvidenceRecord` documents through Keel's action-event sink for every governed decision and invocation (CHR-EVID-001..005, CHR-AGENT-006..008).
- [ ] Expose Charter exception, escalation, and evidence-bundle outputs linked to the affected process/task and runtime context; never require secrets or raw reasoning (CHR-EVID-005..009).
- [ ] Apply Charter information-governance decisions before model/capability access and before retaining prompts, intermediate artifacts, outputs, caches, or external references (CHR-INFO-001..008).

## Conformance

- [ ] Use Charter validator/report contracts through Keel's validation hooks and preserve `CHR-RULE-*`, normative `CHR-*`, and verification-class results (CHR-CONF-005..012).
- [ ] Pass every applicable active core-model fixture. Claim agent-runtime or system-adapter conformance only after Charter activates those profiles with executable rules and valid/invalid fixtures.
