# Information Governance

Status: Draft, unreleased

Charter describes information required or produced by governed work without defining a universal enterprise data model.

## Requirements

- **CHR-INFO-001:** Governed information MUST have an identified business meaning and owning or stewarding authority.
- **CHR-INFO-002:** Information used by an agent MUST declare or inherit classification, permitted use, and access constraints.
- **CHR-INFO-003:** A capability contract MUST expose the minimum information necessary for its business operation.
- **CHR-INFO-004:** Derived information MUST retain provenance sufficient to identify material sources and transformations.
- **CHR-INFO-005:** Retention and deletion requirements MUST apply to action evidence, prompts, intermediate artifacts, caches, and external references as appropriate.
- **CHR-INFO-006:** A binding MUST declare transformations that reduce precision, omit fields, change classifications, or cross governance boundaries.
- **CHR-INFO-007:** An implementation MUST NOT use information for an undeclared purpose solely because an actor can technically access it.
- **CHR-INFO-008:** Conflicting information-governance rules MUST fail closed or follow an explicitly declared precedence policy.

Customer, personal, licensed, and vendor-proprietary information remain governed by their applicable policies and rights outside this specification.
