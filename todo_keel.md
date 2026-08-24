# Keel interfaces required by Charter

Keel remains domain-independent. Charter owns all functional document types and resolution semantics; Keel is the backend framework whose ports Charter's SDK foundations use directly.

## Principal and authorization

- [ ] Replace token/user-id authorization inputs with a resolved principal: stable actor reference, tenant boundary, credential reference, and authentication context. Scopes are not authority (CHR-ID-001..008, CHR-SEC-001..003).
- [ ] Add credential-to-actor and effective-time actor-state resolver interfaces; unresolved or inactive actors fail closed (CHR-ID-003..007).
- [ ] Replace boolean permission checks with a contextual evaluator accepting principal, operation, resource, action time, and opaque domain context (CHR-AUTH-001..002, CHR-AUTH-008..010).
- [ ] Return a structured decision: `allowed|denied|missing|error`, reason, decision/policy references, and constraint results. Preserve authorization, approval, policy, and separation-of-duties as distinct decisions (CHR-AUTH-006..010).
- [ ] Apply the same pre-invocation evaluator port at REST, generated CRUD, table-action, worker, and adapter boundaries; cache keys include every context dimension (CHR-SEC-003, CHR-SEC-006).

## Invocation and evidence

- [ ] Add a generic governed-invocation envelope carrying principal, operation/version, resource context, idempotency key for mutations, and trace context (CHR-CAP-004, CHR-SEC-008).
- [ ] Separate domain outcomes/errors, denied decisions, and transport/platform failures in the result contract (CHR-CAP-005, CHR-SEC-007).
- [ ] Replace integer-user-only audit attribution with stable actor, runtime/execution context, operation, time, outcome, and structured governance decisions (CHR-EVID-001..002, CHR-AGENT-006).
- [ ] Expose an append-only action-event sink with supersession and integrity metadata; delivery/outbox acknowledgement remains separate from the immutable event (CHR-EVID-003..005).
- [ ] Carry stable string references, namespace, tenant boundary, and namespaced extensions without truncation or reinterpretation (CHR-CONF-002..004, CHR-CONF-012).

## Validation

- [ ] Add a generic validator port returning rule id, requirement id, verification class, path, and message; Charter supplies schemas and semantic rules (CHR-CONF-003, CHR-CONF-006..008, CHR-CONF-012).
- [ ] Add a generic conformance-report sink carrying specification/profile versions, rule-set revision, result time, per-rule outcomes, and partial/unsupported features (CHR-CONF-005, CHR-CONF-009..011).
- [ ] Add adapter contract tests proving the active Charter core-model fixtures preserve their exact rule and requirement identifiers.
