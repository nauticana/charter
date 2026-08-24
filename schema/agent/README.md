# Agent Schemas

Draft schemas for the `CHR-ID`, `CHR-AGENT`, and `CHR-AUTH` domains: `human_identity`, `agent_identity`, `agent_definition`, `agent_runtime`, `authority_grant`, `approval`, and `sod_constraint`.

Identity, definition, grant, and approval are separate documents; an agent definition must declare purpose, triggers, inputs, outcomes, capabilities, policies, and escalation conditions.

## Relations

```mermaid
flowchart LR
    AgentDefinition -->|agentIdentityId| AgentIdentity
    AgentDefinition -->|accountable| Position
    AgentDefinition -->|responsibilityIds| Responsibility
    AgentDefinition -->|capabilityIds| CapabilityContract
    AgentRuntime -->|agentIdentityId| AgentIdentity
    AgentRuntime -->|agentDefinitionId| AgentDefinition
    AgentRuntime -->|operator| OrganizationUnit
    AuthorityGrant -->|actor| AgentIdentity
    AuthorityGrant -.->|actor| HumanIdentity
    AuthorityGrant -->|capabilityId| CapabilityContract
    AuthorityGrant -->|delegation.delegator| Position
    Approval -->|approver| HumanIdentity
    Approval -->|responsibilityId| Responsibility
    SodConstraint -->|constrainedActions| CapabilityContract
    HumanIdentity -.->|lifecycleAuthority| Position
    AgentIdentity -.->|lifecycleAuthority| Position

    classDef ext stroke-dasharray: 4 3
    class Position,Responsibility,OrganizationUnit,CapabilityContract ext
```

Dashed nodes belong to other domains.