# Information-Governance Schemas

Schemas for the `CHR-INFO` domain: `information_definition` and `information_governance_policy`.

An information definition states business meaning and ownership and links to explicit governance policies. Policies declare permitted uses, access constraints, retention and deletion rules, precedence, and conflict behavior. These schemas describe governed information without defining a universal enterprise data model: a definition may declare a `shape` of typed properties, identity, and relationships that its data bindings must cover (CHR-INFO-010), and a `factAuthority` ranking the sources of facts about it and bounding their age (CHR-EVID-014, CHR-EVID-015).

## Relations

```mermaid
flowchart LR
    InformationDefinition -->|governancePolicyIds| InformationGovernancePolicy
    InformationGovernancePolicy -->|scope| InformationDefinition
    InformationDefinition -->|ownerOrSteward| OrganizationUnit
    DataBinding -->|informationDefinitionId| InformationDefinition
    InformationDefinition -.->|shape.relationships.targetInformationId| InformationDefinition
    InformationDefinition -.->|factAuthority.ranks.systemProfileId| SystemProfile

    classDef ext stroke-dasharray: 4 3
    class OrganizationUnit,DataBinding,SystemProfile ext
```

Dashed nodes belong to other domains.