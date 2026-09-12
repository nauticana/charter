package authority

import (
	"context"
	"fmt"
	"strings"

	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/temporal"
)

// AbstractEvaluator implements validity, context, and typed-limit evaluation; embed it and supply a GrantSource.
type AbstractEvaluator struct {
	Source      GrantSource
	Limits      LimitEvaluator
	Delegations DelegationPolicy
}

var _ Evaluator = (*AbstractEvaluator)(nil)

func (e *AbstractEvaluator) Evaluate(ctx context.Context, req Request) Decision {
	if e.Source == nil {
		return Decision{Result: Error, Reason: "no grant source"}
	}
	if reason := e.Delegations.Check(req.AuthorityChain, req.At, req.Measures); reason != "" {
		return Decision{Result: Denied, Reason: reason}
	}
	grants, err := e.Source.Grants(ctx, req)
	if err != nil {
		return Decision{Result: Error, Reason: err.Error()}
	}
	if len(grants) == 0 {
		return Decision{Result: Missing, Reason: fmt.Sprintf("no grant for %s on %s", req.Actor.ID, req.CapabilityID.ID)}
	}
	var reasons []string
	for _, g := range grants {
		if reason := e.check(g, req); reason != "" {
			reasons = append(reasons, g.ID+": "+reason)
			continue
		}
		return Decision{GrantRef: &model.Ref{Namespace: g.Namespace, ID: g.ID}, Result: Allowed, Reason: "grant effective within scope and limits"}
	}
	return Decision{Result: Denied, Reason: strings.Join(reasons, "; ")}
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
	return e.Limits.Check(g.Limits, req.Measures)
}
