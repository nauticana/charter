# Evidence Schemas

Schemas for the `CHR-EVID` domain: `evidence_record`, `action_record`, `exception`, `escalation`, and `evidence_bundle`.

Records are categorized (observed fact, agent assertion, human decision, external response, derived conclusion), and corrections supersede rather than replace. Derived conclusions require sources and transformations. Every action record states its `disposition` (executed, business-error, denied, failed, unknown) so stopped attempts are evidenced like executed ones (CHR-EVID-010); executed-action records require structured runtime, authority, and approval evaluations, and `informationEvaluations` records information-governance decisions; an action record carries the `postconditionEvaluations` observed for its capability, each satisfied or violated result resting on observed-fact records (CHR-EVID-011), and a record that settles an earlier unknown attempt names it in `reconcilesActionId` rather than restating it (CHR-EVID-012); evidence bundles declare their assurance profile and integrity method, and a single record may carry an `integrity` digest of its content. An action record may also state its `assignmentBasis` (CHR-ENT-012), the `operatingLimitEvaluations` that stopped or admitted it (CHR-AGENT-011), and the `sodExceptionId` it relied on (CHR-AUTH-012); an evidence record's `assessment` holds a definition evaluation or outcome feedback, recorded as a human decision exactly when its judge is human (CHR-EVID-013).

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
    ActionRecord -->|postconditionEvaluations.evidenceRecordIds| EvidenceRecord
    ActionRecord -.->|reconcilesActionId| ActionRecord
    ActionRecord -.->|sodExceptionId| SodException
    EvidenceRecord -.->|assessment.subject| AgentDefinition
    ExceptionRecord -->|affected| TaskInstance
    ExceptionRecord -->|escalationTarget| Position
    Escalation -->|recipient| Position
    EvidenceBundle -->|subject| ProcessInstance
    EvidenceBundle -->|recordIds| EvidenceRecord
    EvidenceBundle -->|recordIds| ActionRecord

    classDef ext stroke-dasharray: 4 3
    class AgentIdentity,AgentRuntime,Assignment,CapabilityContract,AuthorityGrant,Approval,TaskInstance,Position,ProcessInstance,SodException,AgentDefinition ext
```

Dashed nodes belong to other domains.