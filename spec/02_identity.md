# Human and Agent Identity

Status: Version 1.1.0

Identity establishes who or what participates in Charter-governed activity. Authentication credentials are implementation details bound to, but not identical with, an identity.

## Requirements

- **CHR-ID-001:** Every human and agent identity MUST have a stable identifier within an explicit namespace.
- **CHR-ID-002:** An identity MUST declare its kind without relying on naming conventions.
- **CHR-ID-003:** An implementation MUST distinguish an identity from its credentials, sessions, runtime instances, and organizational assignments.
- **CHR-ID-004:** Identity lifecycle MUST represent at least proposed, active, suspended, and retired states or an explicitly mapped equivalent.
- **CHR-ID-005:** A suspended or retired identity MUST NOT initiate new governed actions.
- **CHR-ID-006:** Reinstatement, replacement, and succession MUST preserve the historical identity referenced by existing evidence.
- **CHR-ID-007:** Credential rotation MUST NOT change the enterprise identity of the actor.
- **CHR-ID-008:** Identity records SHOULD identify the authority responsible for activation, suspension, and retirement.
- **CHR-ID-009:** Evidence for a governed action MUST allow the acting identity's lifecycle state at the action time to be established. The identity's retained lifecycle transitions MUST be ordered and effective-dated sufficiently to establish that state.

## Human and agent distinctions

Human and agent identities share stable identification and audit requirements, but their authentication, lifecycle, accountability, and execution characteristics may differ. A model, process, API client, or runtime instance is not automatically an agent identity.
