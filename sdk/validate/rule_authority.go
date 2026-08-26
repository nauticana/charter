package validate

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nauticana/charter/sdk/authority"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/identity"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/temporal"
)

type ActiveActorRule struct{ AbstractRule }
type GrantEffectiveRule struct{ AbstractRule }
type ApprovalValidRule struct{ AbstractRule }

var (
	_ Rule = ActiveActorRule{}
	_ Rule = GrantEffectiveRule{}
	_ Rule = ApprovalValidRule{}
)

func (r ActiveActorRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, a := range r.actions(c) {
		actor, ok := c.ResolveObject(a.Namespace, a.Actor)
		if !ok {
			out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("actor %s is missing", a.Actor.ID)))
			continue
		}
		state, err := identityStateAt(actor, a.ActionTime)
		if err == nil && state == model.LifecycleActive {
			continue
		}
		if err != nil {
			out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("actor %s lifecycle cannot be established: %v", a.Actor.ID, err)))
			continue
		}
		out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("actor %s is %s", a.Actor.ID, state)))
	}
	return out
}

func identityStateAt(d *model.Document, at time.Time) (string, error) {
	var id struct {
		LifecycleState   string                      `json:"lifecycleState"`
		LifecycleHistory []model.LifecycleTransition `json:"lifecycleHistory"`
	}
	if err := json.Unmarshal(d.Raw, &id); err != nil {
		return "", err
	}
	return identity.Lifecycle{Current: id.LifecycleState, History: id.LifecycleHistory}.StateAt(at)
}

func (r GrantEffectiveRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, a := range r.actions(c) {
		for _, ev := range a.AuthorityEvaluations {
			if ev.Result != "allowed" {
				continue
			}
			if ev.AuthorityGrantID == nil || c.KindOfRef(a.Namespace, *ev.AuthorityGrantID) != model.KindAuthorityGrant {
				out = append(out, r.finding(a.Namespace, a.ID, "allowed without a resolvable grant"))
				continue
			}
			d, _ := c.Resolve(a.Namespace, *ev.AuthorityGrantID)
			g, _ := corpus.Decode[model.AuthorityGrant](d)
			grantActor, grantActorOK := c.ResolveObject(g.Namespace, g.Actor)
			actionActor, actionActorOK := c.ResolveObject(a.Namespace, a.Actor)
			if !grantActorOK || !actionActorOK || grantActor.Namespace != actionActor.Namespace || grantActor.ID != actionActor.ID {
				out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("grant %s actor differs", g.ID)))
			}
			if g.Validity == nil || !temporal.NewPeriod(*g.Validity).Contains(a.ActionTime) {
				out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("grant %s not effective at %s", g.ID, a.ActionTime.Format("2006-01-02"))))
			}
		}
	}
	return out
}

func (r ApprovalValidRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, a := range r.actions(c) {
		for _, ev := range a.ApprovalEvaluations {
			if ev.Result != "approved" {
				continue
			}
			if ev.ApprovalID == nil || c.KindOfRef(a.Namespace, *ev.ApprovalID) != model.KindApproval {
				out = append(out, r.finding(a.Namespace, a.ID, "approved without a resolvable approval"))
				continue
			}
			d, _ := c.Resolve(a.Namespace, *ev.ApprovalID)
			p, _ := corpus.Decode[model.Approval](d)
			if p.ApprovedAction != ev.ApprovedAction {
				out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("approval %s applies to %s, not evaluated action %s", p.ID, p.ApprovedAction, ev.ApprovedAction)))
			}
			if p.MaterialInputsDigest != a.MaterialInputsDigest {
				out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("approval %s material inputs differ", p.ID)))
			}
			for _, subject := range p.SubjectRefs {
				if !containsObjectRef(a.Namespace, a.SubjectRefs, p.Namespace, subject) {
					out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("approval %s subject %s is absent from the action", p.ID, subject.ID)))
				}
			}
			if p.IssuedAt.After(a.ActionTime) {
				out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("approval %s issued after action", p.ID)))
			}
			if p.ValidityMode == "expires-at" && p.ExpiresAt != nil && p.ExpiresAt.Before(a.ActionTime) {
				out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("approval %s expired before action", p.ID)))
			}
		}
	}
	return out
}

func containsObjectRef(actionNamespace string, actionRefs []model.ObjectRef, approvalNamespace string, approvalRef model.ObjectRef) bool {
	for _, actionRef := range actionRefs {
		if actionRef.Kind == approvalRef.Kind && actionRef.ID == approvalRef.ID &&
			effectiveRefNamespace(actionNamespace, actionRef.Namespace) == effectiveRefNamespace(approvalNamespace, approvalRef.Namespace) {
			return true
		}
	}
	return false
}

func effectiveRefNamespace(owner, explicit string) string {
	if explicit != "" {
		return explicit
	}
	return owner
}

// SodApprovalRule finds approvals whose approver performed a constrained action on the same subjects (CHR-AUTH-007).
type SodApprovalRule struct{ AbstractRule }

var _ Rule = SodApprovalRule{}

func (r SodApprovalRule) Validate(c *corpus.Corpus) []Finding {
	constraints := allOf[model.SodConstraint](c, model.KindSodConstraint)
	if len(constraints) == 0 {
		return nil
	}
	var out []Finding
	actions := r.actions(c)
	evaluator := authority.SodEvaluator{Scopes: authority.ExactSodScopeMatcher{}}
	for _, p := range allOf[model.Approval](c, model.KindApproval) {
		approver := corpus.ObjectKeyOf(p.Namespace, p.Approver)
		var performed []authority.SodAction
		for _, a := range actions {
			if a.Actor.Kind == p.Approver.Kind && corpus.ObjectKeyOf(a.Namespace, a.Actor) == approver && authority.Performed(a) {
				performed = append(performed, authority.SodAction{Namespace: a.Namespace, Capability: a.CapabilityID, Scope: authority.SubjectScope(a.Namespace, a.SubjectRefs)})
			}
		}
		proposed := authority.SodAction{Namespace: p.Namespace, Capability: model.Ref{ID: p.ApprovedAction}, Scope: authority.SubjectScope(p.Namespace, p.SubjectRefs)}
		ref, conflict, err := evaluator.Conflict(constraints, performed, proposed)
		if err != nil {
			out = append(out, r.finding(p.Namespace, p.ID, err.Error()))
		} else if conflict {
			out = append(out, r.finding(p.Namespace, p.ID, fmt.Sprintf("approver %s performed an action constrained by %s on the approved subjects", p.Approver.ID, ref.ID)))
		}
	}
	return out
}
