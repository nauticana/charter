# Enterprise Schemas

Schemas for the `CHR-ENT` domain: `enterprise`, `organization_unit`, `position_type`, `position`, `role`, `responsibility`, `assignment`, and `organization_relationship`, sharing `../common.schema.json`.

Objects and relationships are separate records: the unit tree is `parentUnitId`, position classification is `positionTypeId`, position placement is `organizationUnitId`, and every other relation is an explicit `OrganizationRelationship` or `Assignment` document. An assignment may declare `applicability`, which chooses among assignments sharing a target and participation (CHR-ENT-012). Enterprise and process definition documents may cite the evidence their facts rest on in `sourceEvidence` (CHR-INFO-009). See [ADR 0001](../../doc/adr/0001-enterprise-object-relationship-model.md).

## Relations

```mermaid
flowchart LR
    Enterprise
    OrganizationUnit -->|parentUnitId| OrganizationUnit
    OrganizationUnit -->|enterpriseId| Enterprise
    Position -->|organizationUnitId| OrganizationUnit
    Position -->|positionTypeId| PositionType
    Responsibility -->|accountable| Position
    Responsibility -.->|accountable| OrganizationUnit
    Role -->|responsibilityIds| Responsibility
    OrganizationRelationship -->|subject, object| Position
    Assignment -->|target| Position
    Assignment -.->|target| Responsibility
    Assignment -.->|target| Role
    Assignment -.->|target| WorkContext
    Assignment -->|subject| HumanIdentity
    Assignment -.->|subject| AgentIdentity

    classDef ext stroke-dasharray: 4 3
    class HumanIdentity,AgentIdentity,WorkContext ext
```

Dashed nodes belong to other domains or are external kinds.