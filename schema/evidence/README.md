# Evidence Schemas

Draft schemas for the `CHR-EVID` domain: `evidence_record`, `action_record`, `exception`, `escalation`, and `evidence_bundle`.

Records are categorized (observed fact, agent assertion, human decision, external response, derived conclusion), and corrections supersede rather than replace. Derived conclusions require sources and transformations. Executed-action records require structured runtime, authority, and approval evaluations; evidence bundles declare their assurance profile and integrity method.

## Relations

```mermaid
flowchart LR
    EvidenceRecord -.->|supersedes| EvidenceRecord
    ActionRecord -->|actor| AgentIdentity
    ActionRecord -->|runtimeContext.runtimeInstanceId| AgentRuntime
    ActionRecord -->|assignmentId| Assignment
    ActionRecord -->|capabilityId| CapabilityContract
    ActionRecord -->|authorityEvaluations.authorityGrantId| AuthorityGrant
    ActionRecord -->|approvalEvaluations.approvalId| Approval
    ActionRecord -->|evidenceRecordIds| EvidenceRecord
    ExceptionRecord -->|affected| TaskInstance
    ExceptionRecord -->|escalationTarget| Position
    Escalation -->|recipient| Position
    EvidenceBundle -->|subject| ProcessInstance
    EvidenceBundle -->|recordIds| EvidenceRecord
    EvidenceBundle -->|recordIds| ActionRecord

    classDef ext stroke-dasharray: 4 3
    class AgentIdentity,AgentRuntime,Assignment,CapabilityContract,AuthorityGrant,Approval,TaskInstance,Position,ProcessInstance ext
```

Dashed nodes belong to other domains.