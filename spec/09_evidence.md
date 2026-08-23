# Evidence, Audit, Exceptions, and Escalation

Status: Draft, unreleased

Evidence demonstrates what context, authority, decisions, and actions produced an outcome. An audit trail organizes evidence without requiring one storage technology.

## Requirements

- **CHR-EVID-001:** Governed action evidence MUST identify the actor, runtime context, responsibility or assignment, capability, action time, and outcome.
- **CHR-EVID-002:** Evidence for an executed action MUST identify the authority and approvals evaluated for that action.
- **CHR-EVID-003:** Evidence MUST distinguish observed facts, agent-produced assertions, human decisions, external-system responses, and derived conclusions.
- **CHR-EVID-004:** Corrections MUST append or link a superseding record and MUST NOT silently replace historical evidence.
- **CHR-EVID-005:** Evidence integrity and provenance MUST be verifiable to the degree required by the applicable conformance profile.
- **CHR-EVID-006:** An exception MUST identify the violated expectation, affected context, disposition state, and responsible escalation target.
- **CHR-EVID-007:** An escalation MUST identify its reason, recipient, requested decision, urgency, and resulting disposition when known.
- **CHR-EVID-008:** Evidence retention and access MUST follow the information-governance constraints attached to the underlying data and action.
- **CHR-EVID-009:** An implementation or conformance profile MUST NOT require disclosure of sensitive reasoning details or model traces when a sufficient decision rationale and evidence record can be produced without them.

Evidence formats may be centralized, distributed, signed, or externally referenced, provided their declared conformance requirements are met.
