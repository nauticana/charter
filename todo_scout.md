# Scout interfaces required by Charter

Scout is an agent runtime and MCP server. It consumes Charter's functional contracts and may adapt them to Keel's domain-independent ports. Applications or optional bindings supply organization documents through Charter interfaces; Scout does not manage organizations or own Charter document definitions.

## Agent and work context

- [ ] Represent each runtime with references to the exact Charter `AgentIdentity` and versioned `AgentDefinition` it operates, and expose effective-time runtime and lifecycle queries without redefining Charter lifecycle states.
- [ ] Accept work through a context interface carrying resolved Charter process/task, assignment, participation, and organization references. Evaluate one assignment context at a time and never treat an agent as a position occupant.
- [ ] Preserve Charter purpose, accountability, responsibilities, triggers, inputs, outcomes, capabilities, policies, confidence boundaries, and escalation conditions; put Scout-only fields in namespaced extensions.

## Governed execution

- [ ] Resolve and authorize a versioned Charter capability before selecting an MCP tool or binding; tool names, transport scopes, and host trust never grant authority.
- [ ] Expose one invocation/result interface that preserves principal, capability and binding versions, assignment and resource context, inputs, idempotency, declared outcomes, business errors, and transport failures.
- [ ] Consume Charter binding and delivery-policy interfaces for all execution paths and report unsupported or lossy semantics as fail-closed results.

## Decisions, evidence, and conformance

- [ ] Consume Charter authority, approval, policy, separation-of-duties, and information-governance evaluator interfaces as distinct decisions for every applicable read, propose, approve, and execute path.
- [ ] Publish Charter action, evidence, exception, escalation, and evidence-bundle outputs through the configured sink interfaces without requiring secrets or raw reasoning.
- [ ] Preserve Charter rule ids, requirement ids, verification classes, and fixture outcomes through any Keel adapter. Claim a profile only after Charter activates it with executable rules and valid/invalid fixtures.
