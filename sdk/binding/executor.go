package binding

import (
	"context"
	"errors"

	"github.com/nauticana/charter/sdk/model"
)

var (
	ErrOutcomeUnknown = errors.New("external outcome unknown; reconcile before retrying")
	ErrNotExecuted    = errors.New("external operation was not executed")
)

// Request is a capability invocation as a binding sees it: Charter inputs and the contract features the caller relies on.
type Request struct {
	Capability       model.Ref
	ContractVersion  string
	Inputs           any
	IdempotencyKey   string
	RequiredFeatures []string
	SubjectRefs      []model.ObjectRef
}

// Response carries a declared outcome or business error with outputs and external references; transport failures are errors.
type Response struct {
	Outcome           string
	BusinessError     string
	Outputs           any
	ExternalReference string
	EvidenceRecordIDs []model.Ref
}

// Executor realizes a capability through an external system. An error wrapping ErrNotExecuted proves nothing happened;
// any other error means the outcome is unknown.
type Executor interface {
	Execute(ctx context.Context, req Request) (Response, error)
}
