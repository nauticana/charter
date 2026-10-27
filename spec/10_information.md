# Information Governance

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
- **CHR-INFO-009:** An enterprise or process document MAY reference the evidence its modeled facts rest on. It MUST do so when those facts were derived, for example extracted from documents or interviews or proposed by an agent, rather than entered by their accountable owner, and the evidence MUST then include the derived conclusion that `CHR-INFO-004` requires. A fact accepted from a proposal stays distinguishable from an observed one through the categories of its evidence (`CHR-EVID-003`).
- **CHR-INFO-010:** An information definition MAY declare a shape: typed properties, the required properties whose values identify an instance, and named relationships to other information definitions with their cardinality. A data binding to an information definition with a shape MUST map every required property, identity included, and the required properties of every relationship target its paths reach, and every `charterPath` MUST resolve against the shape, each segment but the last naming a relationship whose target has a shape and the last a property. Relationship targets are references under `CHR-CONF-012`.

Customer, personal, licensed, and vendor-proprietary information remain governed by their applicable policies and rights outside this specification.
