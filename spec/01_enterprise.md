# Enterprise Structure and Responsibility

Status: Draft, unreleased

This chapter defines the organizational context in which people and agents act.

## Core concepts

An `Enterprise` is the root governance boundary. An `OrganizationUnit` is a structural node in the enterprise's organization-unit tree. A `Position` is a stable organizational assignment point that belongs to one organization unit and can remain in place as people change. It is not a node in the organization-unit tree. A `Role` groups related responsibilities and authority. A `Responsibility` states an accountable obligation and expected outcome. An `Assignment` connects a human or agent identity to a position, responsibility, or defined work context for an effective period.

## Requirements

- **CHR-ENT-001:** Every organizational object MUST belong to exactly one enterprise governance boundary.
- **CHR-ENT-002:** Every responsibility MUST identify its expected outcome and accountable position or equivalent accountable organizational authority.
- **CHR-ENT-003:** Assignment of work to an agent MUST NOT remove the accountable human or organizational authority unless applicable governance explicitly permits non-human accountability.
- **CHR-ENT-004:** Organization-unit, position, role, responsibility, and assignment identifiers MUST remain distinct even when their display names are equal.
- **CHR-ENT-005:** An implementation MUST support effective dates or lifecycle state sufficient to distinguish current, planned, and retired assignments.
- **CHR-ENT-006:** Responsibility coverage analysis MUST distinguish unassigned work, human-performed work, agent-supported work, and agent-executed work.
- **CHR-ENT-007:** A role assignment MUST NOT implicitly grant every authority associated with similarly named roles in an external system.
- **CHR-ENT-008:** Within an enterprise, organization units MUST form a rooted, acyclic tree in which every non-root organization unit has exactly one parent organization unit.
- **CHR-ENT-009:** A position MUST belong to exactly one organization unit and MUST NOT participate as a parent or child node in the organization-unit tree.

## Relationships

Organization-unit hierarchy, position membership, position-reporting relationships, responsibility ownership, role membership, and work assignments are separate relationships. Implementations MUST NOT infer one solely from another unless an explicit policy defines that inference.
