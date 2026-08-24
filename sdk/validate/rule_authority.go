package validate

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
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
	var identity struct {
		LifecycleState   string                      `json:"lifecycleState"`
		LifecycleHistory []model.LifecycleTransition `json:"lifecycleHistory"`
	}
	if err := json.Unmarshal(d.Raw, &identity); err != nil {
		return "", err
	}
	if len(identity.LifecycleHistory) == 0 {
		return "", fmt.Errorf("lifecycleHistory is missing")
	}
	state := ""
	var previous time.Time
	for _, transition := range identity.LifecycleHistory {
		if transition.EffectiveAt.IsZero() || (!previous.IsZero() && !transition.EffectiveAt.After(previous)) {
			return "", fmt.Errorf("lifecycleHistory is not strictly ordered")
		}
		previous = transition.EffectiveAt
		if !transition.EffectiveAt.After(at) {
			state = transition.State
		}
	}
	if identity.LifecycleState != identity.LifecycleHistory[len(identity.LifecycleHistory)-1].State {
		return "", fmt.Errorf("current lifecycleState differs from latest transition")
	}
	if state == "" {
		return "", fmt.Errorf("no transition is effective at action time")
	}
	return state, nil
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
