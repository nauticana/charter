package authority

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/nauticana/charter/sdk/identity"
	"github.com/nauticana/charter/sdk/model"
)

// AbstractEvaluator implements validity, revocation, context, certification, and typed-limit evaluation; embed it and
// supply a GrantSource. Identities, when set, name each actor's lifecycle authority among an explanation's resolvers.
type AbstractEvaluator struct {
	Source      GrantSource
	Limits      LimitEvaluator
	Delegations DelegationPolicy
	Identities  identity.Provider
}

var _ Evaluator = (*AbstractEvaluator)(nil)

func (e *AbstractEvaluator) Evaluate(ctx context.Context, req Request) Decision {
	d, _, _ := e.evaluate(ctx, req)
	return d
}

// Explain runs the evaluation Evaluate would and reports what decided it, without performing anything. Each unmet
// requirement names who can resolve it: the delegators of the presented chain and of the grants considered, and the
// actor's lifecycle authority.
func (e *AbstractEvaluator) Explain(ctx context.Context, req Request) Explanation {
	_, x, grants := e.evaluate(ctx, req)
	if len(x.Missing) == 0 {
		return x
	}
	resolvers := e.resolvers(ctx, req, grants)
	for i := range x.Missing {
		x.Missing[i].Resolvers = resolvers
	}
	return x
}

// resolvers lists, once each and with explicit namespaces, who can change an evaluation's outcome. An identity that
// cannot be read contributes no lifecycle authority.
func (e *AbstractEvaluator) resolvers(ctx context.Context, req Request, grants []model.AuthorityGrant) []model.ObjectRef {
	var out []model.ObjectRef
	add := func(owner string, r model.ObjectRef) {
		r.Namespace = effectiveNamespace(owner, r.Namespace)
		if r.ID != "" && !slices.Contains(out, r) {
			out = append(out, r)
		}
	}
	for _, hop := range req.AuthorityChain {
		add(req.Namespace, hop.Delegator)
	}
	for _, g := range grants {
		if g.Delegation != nil {
			add(g.Namespace, g.Delegation.Delegator)
		}
	}
	if e.Identities != nil {
		ref := model.Ref{Namespace: req.Actor.Namespace, ID: req.Actor.ID}
		var authority *model.ObjectRef
		switch req.Actor.Kind {
		case model.KindHumanIdentity:
			if h, err := e.Identities.Human(ctx, req.Namespace, ref); err == nil {
				authority = h.LifecycleAuthority
			}
		case model.KindAgentIdentity:
			if a, err := e.Identities.Agent(ctx, req.Namespace, ref); err == nil {
				authority = a.LifecycleAuthority
			}
		}
		if authority != nil {
			add(effectiveNamespace(req.Namespace, req.Actor.Namespace), *authority)
		}
	}
	return out
}

func (e *AbstractEvaluator) evaluate(ctx context.Context, req Request) (Decision, Explanation, []model.AuthorityGrant) {
	x := Explanation{Actor: req.Actor, CapabilityID: req.CapabilityID, ResourceScope: req.ResourceScope, OrganizationalContext: req.OrganizationalContext,
		At: req.At, Basis: []Basis{}}
	var grants []model.AuthorityGrant
	decide := func(d Decision, missing ...string) (Decision, Explanation, []model.AuthorityGrant) {
		x.Decision = d.Result
		for _, m := range missing {
			x.Missing = append(x.Missing, Unmet{Requirement: m})
		}
		return d, x, grants
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
	if g.ResourceScope != req.ResourceScope {
		return "resource scope differs"
	}
	if g.OrganizationalContext != req.OrganizationalContext {
		return "organizational context differs"
	}
	if reason := Lapse(g, req.At); reason != "" {
		return reason
	}
	return e.Limits.Check(g.Limits, req.Measures)
}
