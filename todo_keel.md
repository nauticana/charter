# Keel interfaces required by Charter

Keel remains a domain-independent backend framework. It supplies generic ports that an optional Charter adapter can use; it does not own Charter documents, organization management, actor lifecycle, or authority semantics.

Reviewed against keel v1.2.55: none of the items below is keel work. Keel already exposes the primitives an adapter needs (`port.Principal` + `common` context keys for identity and tenant, `CheckPermission` and `guard.GuardChain` for admission, `port.TableLogger` for row-change audit, `RequestID` for tracing); everything Charter-shaped on top of them — decision semantics, invocation envelopes, event and report schemas, validation rules, fixtures — belongs to the Charter SDK or the adapter. Keel takes a new generic port only when an implemented adapter demonstrates the shape.

## Principal and decisions

- [x] **addressed in the Charter adapter** — `port.Principal` plus the `PartnerID` / `ApiKeyID` / `AuthPrincipal` context keys feed `keel.Session`, `IdentityMap`, and `Caller`, preserving the opaque subject, tenant, credential, authentication claims, and transport scopes without treating scopes as domain authority.
- [x] **addressed in the Charter SDK and adapter** — `authority.Evaluator` carries the actor, capability, resource and organizational context, action time, and measures; its decision distinguishes `allowed`, `denied`, `missing`, and `error` with reasons and grant references. `keel.PermissionGate` and `GuardedInvoker` layer Keel checks behind that decision without widening it.
- [ ] **downstream wiring** — the evaluator and Keel wrapper now exist, but consuming applications still need to inject them at HTTP, generated CRUD, table-action, worker, and adapter boundaries without interpreting their domain context.

## Invocation and evidence

- [x] **addressed in the Charter SDK and adapter** — `capability.Invoker`, `Invocation`, and `Result` carry actor/runtime context, contract version, resource context, idempotency, structured domain outcomes, denials, failures, and evidence references. `keel.GuardedInvoker` adds request correlation and Keel trust guards.
- [x] **addressed in the Charter SDK and adapter** — `evidence.Sink` accepts action, evidence, exception, escalation, and integrity-bearing bundle records. `keel.TableLogStore` appends them with partner/owner scope snapshots, and `PublishingSink` emits the appended document.
- [ ] **partially addressed in the Charter adapter** — `TableLogStore` now snapshots and applies partner/owner scope and preserves JSON documents, but explicit contract fixtures must still prove stable string references, namespaces, and namespaced extensions cross the storage boundary without truncation or reinterpretation.

## Validation and conformance

- [x] **addressed in the Charter SDK** — `validate.Validator` and `validate.Rule` return `Finding` values containing rule id, requirement ids, verification class, namespace/document, path, and message for structural and semantic validation.
- [ ] **not keel** — Charter `conformance/` domain. Expose a conformance-report sink carrying specification/profile versions, implementation version, tested rules, result time, per-rule outcomes, and partial or unsupported features.
- [ ] **not keel** — fixtures live in the adapter repository, where the ports are exercised. Provide adapter contract fixtures proving that rule ids, requirement ids, verification classes, namespaces, and structured outcomes cross Keel ports unchanged.
