package capability

import (
	"context"
	"time"

	"github.com/nauticana/charter/sdk/authority"
	"github.com/nauticana/charter/sdk/model"
)

type Status string

const (
	StatusExecuted      Status = "executed"
	StatusBusinessError Status = "business-error"
	StatusDenied        Status = "denied"
	StatusFailed        Status = "failed"
	StatusUnknown       Status = "unknown"
)

// Invocation is one governed request to use a capability within an execution and assignment context.
type Invocation struct {
	Namespace             string
	EnterpriseID          model.Ref
	Actor                 model.ObjectRef
	Runtime               model.RuntimeContext
	AssignmentID          *model.Ref
	ResponsibilityID      *model.Ref
	CapabilityID          model.Ref
	ContractVersion       string
	SystemProfileID       *model.Ref
	RequiredFeatures      []string
	ResourceScope         string
	OrganizationalContext string
	Measures              map[string]authority.Measure
	// ApprovedAction is the action class an approval must bind; empty means the capability id.
	ApprovedAction       string
	Inputs               any
	MaterialInputsDigest string
	SubjectRefs          []model.ObjectRef
	IdempotencyKey       string
	At                   time.Time
}

// Result is the structured outcome of a governed invocation; Action is the evidence appended for it.
type Result struct {
	Status            Status
	Outcome           string
	BusinessError     string
	Outputs           any
	ExternalReference string
	Reason            string
	Requirement       string
	Authority         authority.Decision
	Approval          authority.ApprovalDecision
	SodConflict       *model.Ref
	Binding           *model.Ref
	Action            *model.ActionRecord
	Err               error
}

type Invoker interface {
	Invoke(ctx context.Context, inv Invocation) Result
}
