# Process and Capability Schemas

Draft schemas for the `CHR-PROC`, `CHR-CAP`, and `CHR-ARCH` domains: `value_stream`, `business_process`, `task`, `process_relationship`, `process_instance`, `task_instance`, `capability_contract`, `architecture_state`, `gap`, and `roadmap_item`.

Definition and instance documents are separate. Execute contracts must declare idempotency; target architecture states must name their decision authority.

## Relations

```mermaid
flowchart LR
    BusinessProcess -->|valueStreamId| ValueStream
    BusinessProcess -.->|parentProcessId| BusinessProcess
    Task -->|processId| BusinessProcess
    Task -->|accountableResponsibilityId| Responsibility
    ProcessRelationship -->|subject, object| Task
    ProcessRelationship -.->|subject, object| BusinessProcess
    ProcessInstance -->|processId| BusinessProcess
    TaskInstance -->|processInstanceId| ProcessInstance
    TaskInstance -->|taskId| Task
    TaskInstance -->|assignmentId| Assignment
    TaskInstance -->|performer| AgentIdentity
    CapabilityContract -->|constraints.sodConstraintIds| SodConstraint
    Gap -->|baselineStateId, targetStateId| ArchitectureState
    RoadmapItem -->|addressesGapIds| Gap
    RoadmapItem -->|owner| OrganizationUnit
    ArchitectureState -->|decisionAuthority| Position

    classDef ext stroke-dasharray: 4 3
    class Responsibility,Assignment,AgentIdentity,SodConstraint,OrganizationUnit,Position ext
```

Dashed nodes belong to other domains.