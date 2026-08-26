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
	AgentIdentityID      Ref       `json:"agentIdentityId"`
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
