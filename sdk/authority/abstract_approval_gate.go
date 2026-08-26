package authority

import (
	"context"
	"fmt"
	"strings"

	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/temporal"
)

// AbstractApprovalGate binds approvals by approved action, material-inputs digest, subjects, limits, and validity mode;
// embed it and supply an ApprovalSource that routes or looks up approvals.
type AbstractApprovalGate struct {
	Source ApprovalSource
	Limits LimitEvaluator
}

var _ ApprovalGate = (*AbstractApprovalGate)(nil)

func (g *AbstractApprovalGate) Evaluate(ctx context.Context, req ApprovalRequest) ApprovalDecision {
	if g.Source == nil {
		return ApprovalDecision{ApprovedAction: req.ApprovedAction, Result: ApprovalError, Reason: "no approval source"}
	}
	approvals, err := g.Source.Approvals(ctx, req)
	if err != nil {
		return ApprovalDecision{ApprovedAction: req.ApprovedAction, Result: ApprovalError, Reason: err.Error()}
	}
	var stale, unbound []string
	for _, p := range approvals {
		if reason := g.binds(p, req); reason != "" {
			unbound = append(unbound, p.ID+": "+reason)
			continue
		}
		if reason := g.timely(p, req); reason != "" {
			stale = append(stale, p.ID+": "+reason)
			continue
		}
		return ApprovalDecision{ApprovalRef: &model.Ref{Namespace: p.Namespace, ID: p.ID}, ApprovedAction: req.ApprovedAction, Result: Approved,
			Reason: "approval binds the action, material inputs, subjects, and limits and is valid at action time"}
	}
	if len(stale) > 0 {
		return ApprovalDecision{ApprovedAction: req.ApprovedAction, Result: ApprovalStale, Reason: strings.Join(stale, "; ")}
	}
	if len(unbound) > 0 {
		return ApprovalDecision{ApprovedAction: req.ApprovedAction, Result: NoApproval, Reason: strings.Join(unbound, "; ")}
	}
	return ApprovalDecision{ApprovedAction: req.ApprovedAction, Result: NoApproval, Reason: "no approval for " + req.ApprovedAction}
}

func (g *AbstractApprovalGate) binds(p model.Approval, req ApprovalRequest) string {
	if p.ApprovedAction != req.ApprovedAction {
		return fmt.Sprintf("approves %s, not %s", p.ApprovedAction, req.ApprovedAction)
	}
	if req.EnterpriseID.ID != "" && (p.EnterpriseID == nil || !sameRef(p.Namespace, *p.EnterpriseID, req.Namespace, req.EnterpriseID)) {
		return "enterprise differs"
	}
	if p.MaterialInputsDigest != req.MaterialInputsDigest {
		return "material inputs differ"
	}
	for _, subject := range p.SubjectRefs {
		if !containsObjectRef(p.Namespace, subject, req.Namespace, req.SubjectRefs) {
			return fmt.Sprintf("subject %s %s is absent from the action", subject.Kind, subject.ID)
		}
	}
	return g.Limits.Check(p.Limits, req.Measures)
}

func (g *AbstractApprovalGate) timely(p model.Approval, req ApprovalRequest) string {
	if p.ValidityMode == model.ValidityExpiresAt && p.ExpiresAt == nil {
		return "expires-at approval without expiry"
	}
	window := temporal.Window{From: p.IssuedAt}
	if p.ValidityMode == model.ValidityExpiresAt {
		window.To = p.ExpiresAt
	}
	if !window.Contains(req.At) {
		return fmt.Sprintf("not valid at %s", req.At.Format("2006-01-02T15:04:05Z07:00"))
	}
	return ""
}

func containsObjectRef(owner string, ref model.ObjectRef, listOwner string, list []model.ObjectRef) bool {
	for _, candidate := range list {
		if sameObjectRef(listOwner, candidate, owner, ref) {
			return true
		}
	}
	return false
}
