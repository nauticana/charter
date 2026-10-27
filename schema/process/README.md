# Process and Capability Schemas

Schemas for the `CHR-PROC`, `CHR-CAP`, and `CHR-ARCH` domains: `value_stream`, `business_process`, `task`, `process_relationship`, `process_instance`, `task_instance`, `capability_contract`, `architecture_state`, `gap`, `roadmap_item`, and `outcome_measure`.

Definition and instance documents are separate. Execute contracts must declare idempotency; target architecture states must name their decision authority. Definition documents may cite their `sourceEvidence` (CHR-INFO-009), and a task instance's participation never exceeds its assignment's level (CHR-PROC-008). An agent performs a measured task above `recommends` only under an assignment that began after each measure in effect was baselined (CHR-PROC-009). A task instance not started by its `dueAt` is escalated to its task's accountable authority (CHR-PROC-010). A capability contract may type its inputs and outputs in `inputSchema` and `outputSchema`, by an information shape or an embedded JSON Schema (CHR-CAP-011), and its preconditions and postconditions may carry a `predicate` in CEL; a precondition with a predicate is an object with an `id` (CHR-CAP-012).

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
    CapabilityContract -.->|inputSchema.informationId, outputSchema.informationId| InformationDefinition
    Gap -->|baselineStateId, targetStateId| ArchitectureState
    RoadmapItem -->|addressesGapIds| Gap
    RoadmapItem -->|owner| OrganizationUnit
    ArchitectureState -->|decisionAuthority| Position
    OutcomeMeasure -->|subject| BusinessProcess
    OutcomeMeasure -.->|subject| Task
    OutcomeMeasure -.->|subject| ValueStream
    OutcomeMeasure -->|baseline.evidenceRecordIds| EvidenceRecord

    classDef ext stroke-dasharray: 4 3
    class Responsibility,Assignment,AgentIdentity,SodConstraint,OrganizationUnit,Position,EvidenceRecord,InformationDefinition ext
```

Dashed nodes belong to other domains.