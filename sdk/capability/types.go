package capability

import (
	"context"
	"time"

	"github.com/nauticana/charter/sdk/authority"
	"github.com/nauticana/charter/sdk/information"
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

// InformationUse declares information an invocation reads or produces and the purpose it serves (CHR-INFO-007).
type InformationUse struct {
	Information model.Ref
	Purpose     string
}

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
	AuthorityChain        model.AuthorityChain
	// ApprovedAction is the action class an approval must bind; empty means the capability id.
	ApprovedAction       string
	Inputs               any
	MaterialInputsDigest string
	SubjectRefs          []model.ObjectRef
	IdempotencyKey       string
	InformationUses      []InformationUse
	At                   time.Time
}

// Result is the structured outcome of a governed invocation; Action is the evidence appended for it, and Exception
// and Escalation the records appended when the invocation had to stop.
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
	Information       []information.Decision
	SodConflict       *model.Ref
	Binding           *model.Ref
	Action            *model.ActionRecord
	Exception         *model.Ref
	Escalation        *model.Ref
	Err               error
}

type Invoker interface {
	Invoke(ctx context.Context, inv Invocation) Result
}
