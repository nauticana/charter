package model

import (
	"encoding/json"
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

const (
	TaskPending   = "pending"
	TaskRunning   = "running"
	TaskSuspended = "suspended"
	TaskCompleted = "completed"
	TaskCancelled = "cancelled"
	TaskFailed    = "failed"
	TaskEscalated = "escalated"
)

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
	DueAt                 *time.Time `json:"dueAt,omitempty"`
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
	ID                     string     `json:"id"`
	Statement              string     `json:"statement"`
	VerificationRequired   bool       `json:"verificationRequired"`
	Outcomes               []string   `json:"outcomes,omitempty"`
	ViolationBusinessError string     `json:"violationBusinessError,omitempty"`
	Predicate              *Predicate `json:"predicate,omitempty"`
}

// DataSchema types a contract's inputs or outputs by an InformationDefinition shape or an embedded JSON Schema (CHR-CAP-011).
type DataSchema struct {
	InformationID *Ref            `json:"informationId,omitempty"`
	JSONSchema    json.RawMessage `json:"jsonSchema,omitempty"`
}

const PredicateCEL = "cel"

// Predicate is a machine-checkable condition over the invocation input and observed facts (CHR-CAP-012).
type Predicate struct {
	Language   string `json:"language"`
	Expression string `json:"expression"`
}

// Precondition is a prose statement, or an identified statement with an optional predicate checked before execution.
type Precondition struct {
	ID        string
	Statement string
	Predicate *Predicate
}

type preconditionObject struct {
	ID        string     `json:"id"`
	Statement string     `json:"statement"`
	Predicate *Predicate `json:"predicate,omitempty"`
}

func (p *Precondition) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*p = Precondition{Statement: s}
		return nil
	}
	var o preconditionObject
	if err := json.Unmarshal(b, &o); err != nil {
		return err
	}
	*p = Precondition(o)
	return nil
}

func (p Precondition) MarshalJSON() ([]byte, error) {
	if p.ID == "" && p.Predicate == nil {
		return json.Marshal(p.Statement)
	}
	return json.Marshal(preconditionObject(p))
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
	InputSchema          *DataSchema           `json:"inputSchema,omitempty"`
	OutputSchema         *DataSchema           `json:"outputSchema,omitempty"`
	Preconditions        []Precondition        `json:"preconditions"`
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

const (
	ImprovementIncrease = "increase"
	ImprovementDecrease = "decrease"
)

// OutcomeMeasure measures the outcome of its subject; currency values are integer minor units (CHR-PROC-009).
type OutcomeMeasure struct {
	Envelope
	Subject          ObjectRef   `json:"subject"`
	Improvement      string      `json:"improvement"`
	Unit             string      `json:"unit,omitempty"`
	Currency         string      `json:"currency,omitempty"`
	CurrencyExponent *int        `json:"currencyExponent,omitempty"`
	Target           json.Number `json:"target,omitempty"`
	Baseline         *Baseline   `json:"baseline,omitempty"`
}

// Baseline is the value observed over a closed window and the evidence it was measured from.
type Baseline struct {
	Value             json.Number `json:"value"`
	Window            Validity    `json:"window"`
	EvidenceRecordIDs []Ref       `json:"evidenceRecordIds"`
}
