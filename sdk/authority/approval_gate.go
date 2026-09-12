package authority

import (
	"context"
	"time"

	"github.com/nauticana/charter/sdk/model"
)

type ApprovalResult string

const (
	Approved      ApprovalResult = model.ApprovalApproved
	ApprovalStale ApprovalResult = model.ApprovalStale
	NoApproval    ApprovalResult = model.ApprovalMissing
	ApprovalError ApprovalResult = "error"
)

// ApprovalRequest describes the action an approval must bind: approver-independent action class, material inputs,
// subjects, time, and measured limits (CHR-AUTH-009).
type ApprovalRequest struct {
	Namespace            string
	EnterpriseID         model.Ref
	Actor                model.ObjectRef
	ApprovedAction       string
	MaterialInputsDigest string
	SubjectRefs          []model.ObjectRef
	At                   time.Time
	Measures             map[string]Measure
	AuthorityChain       model.AuthorityChain
}

// ApprovalDecision is the structured, evidence-ready outcome of an approval evaluation.
type ApprovalDecision struct {
	ApprovalRef    *model.Ref
	ApprovedAction string
	Result         ApprovalResult
	Reason         string
}

type ApprovalGate interface {
	Evaluate(ctx context.Context, req ApprovalRequest) ApprovalDecision
}

// ApprovalSource routes a request to the approvals that may bind it; obtaining approvals is the abstract part of a gate.
type ApprovalSource interface {
	Approvals(ctx context.Context, req ApprovalRequest) ([]model.Approval, error)
}
