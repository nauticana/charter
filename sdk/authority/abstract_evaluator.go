package authority

import (
	"context"
	"fmt"
	"strings"

	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/temporal"
)

// AbstractEvaluator implements validity, context, certification, and typed-limit evaluation; embed it and supply a GrantSource.
type AbstractEvaluator struct {
	Source      GrantSource
	Limits      LimitEvaluator
	Delegations DelegationPolicy
}

var _ Evaluator = (*AbstractEvaluator)(nil)

func (e *AbstractEvaluator) Evaluate(ctx context.Context, req Request) Decision {
	d, _ := e.evaluate(ctx, req)
	return d
}

// Explain runs the evaluation Evaluate would and reports what decided it, without performing anything.
func (e *AbstractEvaluator) Explain(ctx context.Context, req Request) Explanation {
	_, x := e.evaluate(ctx, req)
	return x
}

func (e *AbstractEvaluator) evaluate(ctx context.Context, req Request) (Decision, Explanation) {
	x := Explanation{Actor: req.Actor, CapabilityID: req.CapabilityID, ResourceScope: req.ResourceScope, OrganizationalContext: req.OrganizationalContext,
		At: req.At, Basis: []Basis{}}
	decide := func(d Decision, missing ...string) (Decision, Explanation) {
		x.Decision = d.Result
		for _, m := range missing {
			x.Missing = append(x.Missing, Unmet{Requirement: m})
		}
		return d, x
	}
	if e.Source == nil {
		return decide(Decision{Result: Error, Reason: "no grant source"}, "a grant source")
	}
	if len(req.AuthorityChain) > 0 {
		reason := e.Delegations.Check(req.AuthorityChain, req.At, req.Measures)
		x.Basis = append(x.Basis, basis(BasisDelegation, req.AuthorityChain[0].GrantRef, reason))
		if reason != "" {
			return decide(Decision{Result: Denied, Reason: reason}, "an effective delegation chain")
		}
	}
	grants, err := e.Source.Grants(ctx, req)
	if err != nil {
		return decide(Decision{Result: Error, Reason: err.Error()})
	}
	if len(grants) == 0 {
		return decide(Decision{Result: Missing, Reason: fmt.Sprintf("no grant for %s on %s", req.Actor.ID, req.CapabilityID.ID)}, "an authority grant for the actor and capability")
	}
	var reasons []string
	for _, g := range grants {
		ref := model.Ref{Namespace: g.Namespace, ID: g.ID}
		reason := e.check(g, req)
		x.Basis = append(x.Basis, basis(BasisGrant, ref, reason))
		if reason == "" {
			return decide(Decision{GrantRef: &ref, Result: Allowed, Reason: "grant effective within scope and limits"})
		}
		reasons = append(reasons, g.ID+": "+reason)
	}
	return decide(Decision{Result: Denied, Reason: strings.Join(reasons, "; ")}, "a grant effective within scope and limits")
}

func basis(source string, grant model.Ref, reason string) Basis {
	b := Basis{Source: source, Ref: &model.ObjectRef{Kind: model.KindAuthorityGrant, ID: grant.ID, Namespace: grant.Namespace}, Result: Satisfied, Reason: "effective"}
	if reason != "" {
		b.Result, b.Reason = Unsatisfied, reason
	}
	return b
}

func (e *AbstractEvaluator) check(g model.AuthorityGrant, req Request) string {
	if !sameObjectRef(g.Namespace, g.Actor, req.Namespace, req.Actor) {
		return "actor differs"
	}
	if !sameRef(g.Namespace, g.CapabilityID, req.Namespace, req.CapabilityID) {
		return "capability differs"
	}
	if g.EnterpriseID == nil || !sameRef(g.Namespace, *g.EnterpriseID, req.Namespace, req.EnterpriseID) {
		return "enterprise differs"
	}
	if g.Validity == nil || !temporal.NewPeriod(*g.Validity).Contains(req.At) {
		return "not effective at " + req.At.Format("2006-01-02")
	}
	if g.ResourceScope != req.ResourceScope {
		return "resource scope differs"
	}
	if g.OrganizationalContext != req.OrganizationalContext {
		return "organizational context differs"
	}
	if reason := CertificationLapse(g, req.At); reason != "" {
		return reason
	}
	return e.Limits.Check(g.Limits, req.Measures)
}
