package model

import (
	"slices"
	"time"
)

type ValueStream struct {
	Envelope
	BusinessOutcome string `json:"businessOutcome"`
}

type BusinessProcess struct {
	Envelope
	// ProcessDefinitionVersion is the version a ProcessInstance must name to run this process.
	ProcessDefinitionVersion string   `json:"processDefinitionVersion"`
	BusinessOutcome          string   `json:"businessOutcome"`
	ValueStreamID            *Ref     `json:"valueStreamId,omitempty"`
	ParentProcessID          *Ref     `json:"parentProcessId,omitempty"`
	Aliases                  []string `json:"aliases,omitempty"`
}

type Task struct {
	Envelope
	// TaskDefinitionVersion is the version a TaskInstance must name to run this task.
	TaskDefinitionVersion       string   `json:"taskDefinitionVersion"`
	ProcessID                   Ref      `json:"processId"`
	BusinessOutcome             string   `json:"businessOutcome"`
	AccountableResponsibilityID Ref      `json:"accountableResponsibilityId"`
	PermittedPerformerKinds     []string `json:"permittedPerformerKinds"`
	PermittedParticipation      []string `json:"permittedParticipation,omitempty"`
}

const (
	RelationshipParentChild = "parent-child"
	RelationshipPrecedes    = "precedes"
	RelationshipDependsOn   = "depends-on"
)

type ProcessRelationship struct {
	Envelope
	RelationshipType string    `json:"relationshipType"`
	Subject          ObjectRef `json:"subject"`
	Object           ObjectRef `json:"object"`
}

type ProcessInstance struct {
	Envelope
	ProcessID                Ref         `json:"processId"`
	ProcessDefinitionVersion string      `json:"processDefinitionVersion"`
	State                    string      `json:"state"`
	CreatedAt                time.Time   `json:"createdAt"`
	StartedAt                *time.Time  `json:"startedAt,omitempty"`
	EndedAt                  *time.Time  `json:"endedAt,omitempty"`
	ContextRefs              []ObjectRef `json:"contextRefs,omitempty"`
}

type TaskInstance struct {
	Envelope
	ProcessInstanceID     Ref        `json:"processInstanceId"`
	TaskID                Ref        `json:"taskId"`
	TaskDefinitionVersion string     `json:"taskDefinitionVersion"`
	Performer             ObjectRef  `json:"performer"`
	AssignmentID          Ref        `json:"assignmentId"`
	Participation         string     `json:"participation"`
	State                 string     `json:"state"`
	CreatedAt             time.Time  `json:"createdAt"`
	StartedAt             *time.Time `json:"startedAt,omitempty"`
	EndedAt               *time.Time `json:"endedAt,omitempty"`
	Outcome               string     `json:"outcome,omitempty"`
}

type CapabilityConstraints struct {
	ApprovalRequired bool    `json:"approvalRequired"`
	SodConstraintIDs []Ref   `json:"sodConstraintIds"`
	Limits           []Limit `json:"limits"`
}

type Idempotency struct {
	Mutating       bool   `json:"mutating"`
	RetrySemantics string `json:"retrySemantics"`
	// RetryWhenEffectAbsent declares that the mutation may be sent again once its effect is observed absent (CHR-CAP-010).
	RetryWhenEffectAbsent bool `json:"retryWhenEffectAbsent,omitempty"`
}

// Postcondition is an externally observable effect of a capability. Outcomes limits it to those declared outcomes;
// empty applies it to every outcome (CHR-CAP-009).
type Postcondition struct {
	ID                     string   `json:"id"`
	Statement              string   `json:"statement"`
	VerificationRequired   bool     `json:"verificationRequired"`
	Outcomes               []string `json:"outcomes,omitempty"`
	ViolationBusinessError string   `json:"violationBusinessError,omitempty"`
}

// AppliesTo reports whether the postcondition must hold for a declared outcome; an empty outcome matches every postcondition.
func (p Postcondition) AppliesTo(outcome string) bool {
	return outcome == "" || len(p.Outcomes) == 0 || slices.Contains(p.Outcomes, outcome)
}

type CapabilityContract struct {
	Envelope
	Purpose              string                `json:"purpose"`
	OperationClass       string                `json:"operationClass"`
	ContractVersion      string                `json:"contractVersion"`
	Inputs               []string              `json:"inputs"`
	Outputs              []string              `json:"outputs"`
	Preconditions        []string              `json:"preconditions"`
	Outcomes             []string              `json:"outcomes"`
	BusinessErrors       []string              `json:"businessErrors"`
	RequiredAuthority    string                `json:"requiredAuthority"`
	Constraints          CapabilityConstraints `json:"constraints"`
	Idempotency          Idempotency           `json:"idempotency"`
	EvidenceRequirements []string              `json:"evidenceRequirements"`
	Postconditions       []Postcondition       `json:"postconditions,omitempty"`
}

// RequiredPostconditions are the postconditions that must be verified before outcome may be recorded as executed.
func (c CapabilityContract) RequiredPostconditions(outcome string) []Postcondition {
	var out []Postcondition
	for _, p := range c.Postconditions {
		if p.VerificationRequired && p.AppliesTo(outcome) {
			out = append(out, p)
		}
	}
	return out
}

const (
	StateBaseline = "baseline"
	StateTarget   = "target"
)

type ArchitectureState struct {
	Envelope
	StateType         string     `json:"stateType"`
	DecisionAuthority *ObjectRef `json:"decisionAuthority,omitempty"`
}

type Gap struct {
	Envelope
	BaselineStateID  Ref         `json:"baselineStateId"`
	TargetStateID    Ref         `json:"targetStateId"`
	Classification   string      `json:"classification"`
	ComparedElements []ObjectRef `json:"comparedElements,omitempty"`
}

type RoadmapItem struct {
	Envelope
	AddressesGapIDs []Ref     `json:"addressesGapIds"`
	Owner           ObjectRef `json:"owner"`
	Dependencies    []string  `json:"dependencies,omitempty"`
	Status          string    `json:"status"`
}
