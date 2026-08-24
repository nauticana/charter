# Keel interfaces required by Charter

Keel remains a domain-independent backend framework. It supplies generic ports that an optional Charter adapter can use; it does not own Charter documents, organization management, actor lifecycle, or authority semantics.

## Principal and decisions

- [ ] Expose an authenticated-principal interface carrying a stable opaque subject reference, tenant boundary, credential reference, and authentication context. Do not equate transport scopes with domain authority.
- [ ] Expose a contextual decision-evaluator port whose request carries the principal, operation, resource, action time, and opaque domain context, and whose response distinguishes `allowed`, `denied`, `missing`, and `error` with stable reason and decision references.
- [ ] Make the same evaluator port available at HTTP, generated CRUD, table-action, worker, and adapter boundaries without interpreting its domain context.

## Invocation and evidence

- [ ] Expose governed invocation/result interfaces carrying principal, versioned operation, resource context, idempotency and trace references, while keeping domain outcomes, denials, and platform failures distinct.
- [ ] Expose an append-only action-event sink carrying stable actor and execution references, operation, time, outcome, structured decisions, supersession, and integrity metadata.
- [ ] Preserve stable string references, namespaces, tenant boundaries, and namespaced extensions without truncation or reinterpretation.

## Validation and conformance

- [ ] Expose a validator port returning rule id, requirement id, verification class, path, and message; the adapter supplies Charter schemas and semantic rules.
- [ ] Expose a conformance-report sink carrying specification/profile versions, implementation version, tested rules, result time, per-rule outcomes, and partial or unsupported features.
- [ ] Provide adapter contract fixtures proving that rule ids, requirement ids, verification classes, namespaces, and structured outcomes cross Keel ports unchanged.
