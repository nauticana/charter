# Scout interfaces required by Charter

Scout is an agent runtime and MCP server. It consumes Charter's functional contracts and may adapt them to Keel's domain-independent ports. Applications or optional bindings supply organization documents through Charter interfaces; Scout does not manage organizations or own Charter document definitions.

Reviewed on 2026-08-25 against the Charter SDK: every item remains Scout's work and each has the Charter contract it consumes named beside it. Nothing is blocked on Charter any more: the information-governance gate, exception and escalation output, event delivery contract, and evidence redaction all landed in the SDK.

## Agent and work context

- [ ] Represent each runtime with references to the exact Charter `AgentIdentity` and versioned `AgentDefinition` it operates, and expose effective-time runtime and lifecycle queries without redefining Charter lifecycle states. Charter: `model.AgentRuntime`, `agent.Provider`, `identity.Resolver.ActiveAt`.
- [ ] Accept work through a context interface carrying resolved Charter process/task, assignment, participation, and organization references. Evaluate one assignment context at a time and never treat an agent as a position occupant. Charter: `agent.ExecutionContext` + `agent.BaseAdmission`, `process.BaseContextResolver`; `keel.RuntimeContext` supplies the execution-context id.
- [ ] Preserve Charter purpose, accountability, responsibilities, triggers, inputs, outcomes, capabilities, policies, confidence boundaries, and escalation conditions; put Scout-only fields in namespaced extensions. Charter: `model.AgentDefinition` with `Envelope.Extensions`; CHR-RULE-CONF-001 rejects undeclared non-namespaced properties structurally.

## Governed execution

- [ ] Resolve and authorize a versioned Charter capability before selecting an MCP tool or binding; tool names, transport scopes, and host trust never grant authority. Charter: `capability.AbstractInvoker` (contract, version, lifecycle, authority, approval, SoD, binding gates); wrap it in `keel.GuardedInvoker` and `PermissionGate`.
- [ ] Expose one invocation/result interface that preserves principal, capability and binding versions, assignment and resource context, inputs, idempotency, declared outcomes, business errors, and transport failures. Charter: `capability.Invocation` / `Result`; MCP tools become `binding.Executor` realizations.
- [ ] Consume Charter binding and delivery-policy interfaces for all execution paths and report unsupported or lossy semantics as fail-closed results. Charter: `binding.Features.Require`, `binding.AbstractBinding` for execution; `binding.AbstractEventConsumer` with a `DeliveryLedger` for event delivery, and `agent.Triggered` to match the delivered trigger against the definition.

## Decisions, evidence, and conformance

- [ ] Consume Charter authority, approval, policy, separation-of-duties, and information-governance evaluator interfaces as distinct decisions for every applicable read, propose, approve, and execute path. Charter: `authority.Evaluator`, `authority.ApprovalGate`, `capability.SodChecker`, and `information.Evaluator` through `Invocation.InformationUses`; the invoker records every decision in the action record (`authorityEvaluations`, `approvalEvaluations`, `informationEvaluations`, `disposition`).
- [ ] Publish Charter action, evidence, exception, escalation, and evidence-bundle outputs through the configured sink interfaces without requiring secrets or raw reasoning. Charter: `evidence.Sink` via `keel.TableLogStore` + `PublishingSink`; `capability.BaseEscalator` produces the exception and escalation for stopped actions, `evidence.BaseRedactor` keeps secrets out, and `AbstractSink.Assemble` builds the bundle for a process instance.
- [ ] Preserve Charter rule ids, requirement ids, verification classes, and fixture outcomes through any Keel adapter. Claim a profile only after Charter activates it with executable rules and valid/invalid fixtures. Charter: all three profiles are active with 20 semantic and 12 behavioral rules; Scout claims agent-runtime by implementing `validate.Subject` (and system-adapter for its MCP tools by implementing `validate.AdapterSubject`) and running `charter claim` against them, which names the implementation and reports unexercised rules as not tested (CHR-CONF-014).
