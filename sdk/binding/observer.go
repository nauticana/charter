package binding

import (
	"context"

	"github.com/nauticana/charter/sdk/model"
)

// ObservationRequest asks for the external state a capability's postconditions describe, or, before execution, the
// facts its predicate preconditions read. Outcome is the declared outcome the mutation reported, empty when that
// outcome is unknown.
type ObservationRequest struct {
	Capability        model.Ref
	ContractVersion   string
	Inputs            any
	IdempotencyKey    string
	SubjectRefs       []model.ObjectRef
	ExternalReference string
	Outcome           string
	Postconditions    []model.Postcondition
	Preconditions     []model.Precondition
}

// Observation is the external state read independently of the mutation's response. Outcome names the declared outcome
// the observed state corresponds to; reconciliation of an unknown attempt relies on it. Facts are the named values
// predicates read, stated by the observed-fact records EvidenceRecordIDs names.
type Observation struct {
	Outcome           string
	Evaluations       []model.PostconditionEvaluation
	Facts             map[string]any
	EvidenceRecordIDs []model.Ref
}

// Observer reads the effect of a capability from the external system and records what it saw as observed-fact
// evidence. It never mutates; an error means the state could not be observed.
type Observer interface {
	Observe(ctx context.Context, req ObservationRequest) (Observation, error)
}
