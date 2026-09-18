# Security Requirements

Status: Version 1.1.0

Charter defines security properties that preserve identity, authority, information constraints, and evidence across implementations. It does not certify an implementation as secure.

## Threat model

Implementations should assume that inputs may be malformed or adversarial, external systems may fail or return inconsistent results, credentials may be revoked, agents may make incorrect decisions, and instructions or retrieved content may attempt to influence behavior outside granted authority.

## Requirements

- **CHR-SEC-001:** Authentication MUST establish the actor or system identity before governed authority is evaluated.
- **CHR-SEC-002:** Credentials MUST be stored, transmitted, rotated, and revoked independently of the stable Charter identity.
- **CHR-SEC-003:** Authorization MUST be enforced at the action boundary and MUST NOT rely solely on an agent's self-reported compliance.
- **CHR-SEC-004:** Untrusted instructions and retrieved content MUST NOT expand an actor's authority or override higher-priority policy.
- **CHR-SEC-005:** Implementations MUST isolate secrets from prompts, evidence, logs, and outputs unless their inclusion is explicitly required and protected.
- **CHR-SEC-006:** Cross-enterprise and cross-tenant access MUST be denied unless explicitly authorized for both the actor and information involved.
- **CHR-SEC-007:** Security-relevant failures MUST be observable and MUST NOT be silently converted into successful outcomes.
- **CHR-SEC-008:** Replay, duplicate execution, and partial failure risks MUST be addressed for mutating capabilities.
- **CHR-SEC-009:** Implementations MUST provide a means to suspend an identity or capability and prevent new actions within the declared enforcement latency.
- **CHR-SEC-010:** Security evidence MUST be protected against unauthorized alteration and access.

Conformance demonstrates that declared controls and behaviors are present; it is not a substitute for deployment-specific threat analysis or independent security assessment.
