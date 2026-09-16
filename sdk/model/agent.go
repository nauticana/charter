package model

import "time"

type HumanIdentity struct {
	Envelope
	LifecycleAuthority *ObjectRef            `json:"lifecycleAuthority,omitempty"`
	LifecycleHistory   []LifecycleTransition `json:"lifecycleHistory,omitempty"`
}

type AgentIdentity struct {
	Envelope
	LifecycleAuthority *ObjectRef            `json:"lifecycleAuthority,omitempty"`
	LifecycleHistory   []LifecycleTransition `json:"lifecycleHistory,omitempty"`
}

type AgentDefinition struct {
	Envelope
	AgentIdentityID Ref `json:"agentIdentityId"`
	// DefinitionVersion is the version an AgentRuntime must name to operate this definition.
	DefinitionVersion    string    `json:"definitionVersion"`
	Purpose              string    `json:"purpose"`
	Accountable          ObjectRef `json:"accountable"`
	ResponsibilityIDs    []Ref     `json:"responsibilityIds"`
	Triggers             []string  `json:"triggers"`
	RequiredInputs       []string  `json:"requiredInputs"`
	ExpectedOutcomes     []string  `json:"expectedOutcomes"`
	CapabilityIDs        []Ref     `json:"capabilityIds"`
	Policies             []string  `json:"policies"`
	EscalationConditions []string  `json:"escalationConditions"`
	ConfidenceBoundaries []string  `json:"confidenceBoundaries"`
}

type AgentRuntime struct {
	Envelope
	AgentIdentityID   Ref       `json:"agentIdentityId"`
	AgentDefinitionID Ref       `json:"agentDefinitionId"`
	DefinitionVersion string    `json:"definitionVersion"`
	Operator          ObjectRef `json:"operator"`
}

type Delegation struct {
	Delegator            ObjectRef `json:"delegator"`
	MaxRedelegationDepth int       `json:"maxRedelegationDepth"`
}

// AuthorityChain is the presented delegation path, ordered from the immediate delegator to the original authority,
// matching the nested actor shape of RFC 8693. It carries authority references and bounds, never credentials.
type AuthorityChain []AuthorityHop

// AuthorityHop is one verified delegation step and the bounds it conveys to the next actor.
type AuthorityHop struct {
	GrantRef         Ref       `json:"grantRef"`
	Delegator        ObjectRef `json:"delegator"`
	RemainingDepth   int       `json:"remainingDepth"`
	Limits           []Limit   `json:"limits,omitempty"`
	Validity         *Validity `json:"validity"`
	ApprovalRequired bool      `json:"approvalRequired"`
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

const (
	ValidityExpiresAt = "expires-at"
	ValidityOpenEnded = "open-ended"
)

type Approval struct {
	Envelope
	Approver               ObjectRef   `json:"approver"`
	ResponsibilityID       *Ref        `json:"responsibilityId,omitempty"`
	ApprovedAction         string      `json:"approvedAction"`
	SubjectRefs            []ObjectRef `json:"subjectRefs,omitempty"`
	MaterialInputsDigest   string      `json:"materialInputsDigest"`
	Limits                 []Limit     `json:"limits"`
	Conditions             []string    `json:"conditions"`
	IssuedAt               time.Time   `json:"issuedAt"`
	ValidityMode           string      `json:"validityMode"`
	ExpiresAt              *time.Time  `json:"expiresAt,omitempty"`
	OpenEndedJustification string      `json:"openEndedJustification,omitempty"`
}

type SodConstraint struct {
	Envelope
	ConstrainedActions []Ref  `json:"constrainedActions"`
	Scope              string `json:"scope,omitempty"`
}
