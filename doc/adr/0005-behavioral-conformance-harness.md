# 0005: Runtime-behavioral conformance harness

Status: Proposed
Date: 2026-08-25

## Context

The agent-runtime and system-adapter profiles verify behavior that static documents cannot show: decisions at the action boundary, replay handling, error classification, and evidence. CHR-CONF-011 requires runtime-behavioral results to be distinguishable from structural and semantic ones, and `spec/12_conformance.md` activates a profile only when executable rules and fixtures exist. The fixtures must remain language- and vendor-neutral while the reference harness is Go.

## Decision

- A behavioral fixture is a `fixture.yaml` carrying a `scenario` (format in `schema/conformance/fixture_descriptor.schema.json`): a base context, invocation, or request that steps override field by field, each step with a scripted `transport` or `vendor` reply and an `expect` block naming the status or result, requirement id, evaluation results, call counts, and whether evidence was recorded.
- The implementation under test is a `validate.Subject` for runtimes (`Compose(documents, transport)` yields admission, invoker, and evidence read-back) or a `validate.AdapterSubject` for adapters (`Realize(documents, binding, vendor)` yields an executor). The harness owns the transport or vendor endpoint so external outcomes are scripted, and judges only the subject's decisions and mappings.
- `expected: pass` scenarios must be carried out and `expected: fail` scenarios must be refused; a fixture passes when the subject matches its expectations in either case. This maps CHR-CONF-007's valid and invalid fixture classes onto behavior without inventing a third result class. Adapter results are classified as outcome, business-error, unknown, or not-executed; an unclassified failure never conforms (CHR-SEC-007).
- One generic executor runs every behavioral rule; a rule's semantics are its scenarios, and `BehavioralRule.Kind` selects the subject it needs. The SDK is the default subject and adapter (`ReferenceSubject`, `ReferenceAdapter`); with no subject, behavioral rules report `not-tested` and the claim is `partial` (CHR-CONF-009).
- The reference adapter speaks a neutral vendor protocol (a `status` naming a declared outcome plus a reference; an error text naming a declared business error). Downstream adapters implement the interface over their vendor's shape and ship scenarios in the same format.
- `manifest.yaml`, rule definitions, and fixture descriptors are validated against their format schemas before use, so a malformed scenario fails with a location rather than a misread.

## Consequences

Any Go runtime or adapter runs the whole manifest by implementing one interface; a non-Go implementation reuses the YAML with its own harness. Scenarios are written against the Harbor documents, so they double as executable documentation of the example. Writing the first scenarios exposed a missing lifecycle gate in the invoker (ADR 0003). Because a rule's meaning lives in its scenarios, reviewing a rule means reviewing its two fixtures.

## Specification impact

The scenario semantics (carry out versus refuse, response classes) are documented in `conformance/README.md` and are non-normative. A future release of the conformance chapter may adopt them normatively.
