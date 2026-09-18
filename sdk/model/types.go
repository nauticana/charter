package model

import "time"

type Kind string

const (
	KindEnterprise                  Kind = "Enterprise"
	KindOrganizationUnit            Kind = "OrganizationUnit"
	KindPositionType                Kind = "PositionType"
	KindPosition                    Kind = "Position"
	KindRole                        Kind = "Role"
	KindResponsibility              Kind = "Responsibility"
	KindAssignment                  Kind = "Assignment"
	KindOrganizationRelationship    Kind = "OrganizationRelationship"
	KindHumanIdentity               Kind = "HumanIdentity"
	KindAgentIdentity               Kind = "AgentIdentity"
	KindAgentDefinition             Kind = "AgentDefinition"
	KindAgentRuntime                Kind = "AgentRuntime"
	KindAuthorityGrant              Kind = "AuthorityGrant"
	KindApproval                    Kind = "Approval"
	KindSodConstraint               Kind = "SodConstraint"
	KindValueStream                 Kind = "ValueStream"
	KindBusinessProcess             Kind = "BusinessProcess"
	KindTask                        Kind = "Task"
	KindProcessRelationship         Kind = "ProcessRelationship"
	KindProcessInstance             Kind = "ProcessInstance"
	KindTaskInstance                Kind = "TaskInstance"
	KindCapabilityContract          Kind = "CapabilityContract"
	KindArchitectureState           Kind = "ArchitectureState"
	KindGap                         Kind = "Gap"
	KindRoadmapItem                 Kind = "RoadmapItem"
	KindEnterpriseSystem            Kind = "EnterpriseSystem"
	KindSystemProfile               Kind = "SystemProfile"
	KindCapabilityBinding           Kind = "CapabilityBinding"
	KindDataBinding                 Kind = "DataBinding"
	KindAuthorityBinding            Kind = "AuthorityBinding"
	KindEventBinding                Kind = "EventBinding"
	KindBindingConformance          Kind = "BindingConformance"
	KindActionRecord                Kind = "ActionRecord"
	KindEvidenceRecord              Kind = "EvidenceRecord"
	KindEvidenceBundle              Kind = "EvidenceBundle"
	KindExceptionRecord             Kind = "ExceptionRecord"
	KindEscalation                  Kind = "Escalation"
	KindInformationDefinition       Kind = "InformationDefinition"
	KindInformationGovernancePolicy Kind = "InformationGovernancePolicy"
	KindMediationProfile            Kind = "MediationProfile"
	KindMediationDecision           Kind = "MediationDecision"
	KindConformanceClaim            Kind = "ConformanceClaim"
)

const (
	LifecycleProposed  = "proposed"
	LifecyclePlanned   = "planned"
	LifecycleActive    = "active"
	LifecycleSuspended = "suspended"
	LifecycleRetired   = "retired"
)

const (
	ParticipationOccupies   = "occupies"
	ParticipationSupports   = "supports"
	ParticipationObserves   = "observes"
	ParticipationRecommends = "recommends"
	ParticipationPrepares   = "prepares"
	ParticipationApproves   = "approves"
	ParticipationPerforms   = "performs"
	ParticipationExecutes   = "executes"
)

const (
	PerformerHuman = "human"
	PerformerAgent = "agent"
)

const (
	OperationRead    = "read"
	OperationPropose = "propose"
	OperationApprove = "approve"
	OperationExecute = "execute"
)

const (
	SupportSupported   = "supported"
	SupportPartial     = "partial"
	SupportUnsupported = "unsupported"
)

const (
	ResultConforming    = "conforming"
	ResultPartial       = "partial"
	ResultNonconforming = "nonconforming"
)

const (
	VerificationStructural = "structural"
	VerificationSemantic   = "semantic"
	VerificationBehavioral = "runtime-behavioral"
)

const (
	RulePass      = "pass"
	RuleFail      = "fail"
	RuleNotTested = "not-tested"
)

type LifecycleTransition struct {
	State       string     `json:"state"`
	EffectiveAt time.Time  `json:"effectiveAt"`
	Authority   *ObjectRef `json:"authority,omitempty"`
}

type ObjectRef struct {
	Kind      Kind   `json:"kind"`
	ID        string `json:"id"`
	Namespace string `json:"namespace,omitempty"`
	External  bool   `json:"external,omitempty"`
}

// FeatureSupport declares how one contract feature is supported by a binding or evaluated by a conformance check.
type FeatureSupport struct {
	Feature string `json:"feature"`
	Support string `json:"support"`
	Notes   string `json:"notes,omitempty"`
}
