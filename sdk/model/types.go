package model

import "time"

const (
	KindEnterprise               Kind = "Enterprise"
	KindOrganizationUnit         Kind = "OrganizationUnit"
	KindPositionType             Kind = "PositionType"
	KindPosition                 Kind = "Position"
	KindRole                     Kind = "Role"
	KindResponsibility           Kind = "Responsibility"
	KindAssignment               Kind = "Assignment"
	KindOrganizationRelationship Kind = "OrganizationRelationship"
	KindHumanIdentity            Kind = "HumanIdentity"
	KindAgentIdentity            Kind = "AgentIdentity"
	KindAgentDefinition          Kind = "AgentDefinition"
	KindAgentRuntime             Kind = "AgentRuntime"
	KindAuthorityGrant           Kind = "AuthorityGrant"
	KindApproval                 Kind = "Approval"
	KindSodConstraint            Kind = "SodConstraint"
	KindCapabilityContract       Kind = "CapabilityContract"
	KindActionRecord             Kind = "ActionRecord"
	KindEvidenceRecord           Kind = "EvidenceRecord"
)

const (
	LifecycleProposed  = "proposed"
	LifecyclePlanned   = "planned"
	LifecycleActive    = "active"
	LifecycleSuspended = "suspended"
	LifecycleRetired   = "retired"
)

const ParticipationOccupies = "occupies"

type Kind string

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

type OrganizationUnit struct {
	Envelope
	ParentUnitID *Ref `json:"parentUnitId,omitempty"`
}

type Position struct {
	Envelope
	OrganizationUnitID Ref  `json:"organizationUnitId"`
	PositionTypeID     *Ref `json:"positionTypeId,omitempty"`
}

type Assignment struct {
	Envelope
	Subject       ObjectRef `json:"subject"`
	Target        ObjectRef `json:"target"`
	Participation string    `json:"participation"`
	Condition     string    `json:"condition,omitempty"`
	Reason        string    `json:"reason,omitempty"`
}

type Delegation struct {
	Delegator            ObjectRef `json:"delegator"`
	MaxRedelegationDepth int       `json:"maxRedelegationDepth"`
}

type AuthorityGrant struct {
	Envelope
	Actor                 ObjectRef   `json:"actor"`
	CapabilityID          Ref         `json:"capabilityId"`
	ResourceScope         string      `json:"resourceScope"`
	OrganizationalContext string      `json:"organizationalContext"`
	Limits                []Limit     `json:"limits"`
	Delegation            *Delegation `json:"delegation,omitempty"`
}

type Approval struct {
	Envelope
	Approver             ObjectRef   `json:"approver"`
	ResponsibilityID     *Ref        `json:"responsibilityId,omitempty"`
	ApprovedAction       string      `json:"approvedAction"`
	SubjectRefs          []ObjectRef `json:"subjectRefs,omitempty"`
	MaterialInputsDigest string      `json:"materialInputsDigest"`
	Limits               []Limit     `json:"limits"`
	Conditions           []string    `json:"conditions"`
	IssuedAt             time.Time   `json:"issuedAt"`
	ValidityMode         string      `json:"validityMode"`
	ExpiresAt            *time.Time  `json:"expiresAt,omitempty"`
}

type SodConstraint struct {
	Envelope
	ConstrainedActions []Ref  `json:"constrainedActions"`
	Scope              string `json:"scope,omitempty"`
}

type RuntimeContext struct {
	RuntimeInstanceID  Ref    `json:"runtimeInstanceId"`
	ExecutionContextID string `json:"executionContextId"`
}

type AuthorityEvaluation struct {
	AuthorityGrantID *Ref   `json:"authorityGrantId,omitempty"`
	Result           string `json:"result"`
	Reason           string `json:"reason"`
}

type ApprovalEvaluation struct {
	ApprovalID     *Ref   `json:"approvalId,omitempty"`
	ApprovedAction string `json:"approvedAction,omitempty"`
	Result         string `json:"result"`
	Reason         string `json:"reason"`
}

type ActionRecord struct {
	Envelope
	Actor                ObjectRef             `json:"actor"`
	RuntimeContext       RuntimeContext        `json:"runtimeContext"`
	AssignmentID         *Ref                  `json:"assignmentId,omitempty"`
	ResponsibilityID     *Ref                  `json:"responsibilityId,omitempty"`
	CapabilityID         Ref                   `json:"capabilityId"`
	MaterialInputsDigest string                `json:"materialInputsDigest,omitempty"`
	SubjectRefs          []ObjectRef           `json:"subjectRefs,omitempty"`
	OperationClass       string                `json:"operationClass"`
	ActionTime           time.Time             `json:"actionTime"`
	Outcome              string                `json:"outcome"`
	AuthorityEvaluations []AuthorityEvaluation `json:"authorityEvaluations,omitempty"`
	ApprovalEvaluations  []ApprovalEvaluation  `json:"approvalEvaluations,omitempty"`
}
