package capability

import (
	"context"
	"errors"

	"github.com/nauticana/charter/sdk/authority"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/model"
)

// SodChecker finds the separation-of-duties constraint an invocation would violate (CHR-AUTH-007).
type SodChecker interface {
	Conflict(ctx context.Context, inv Invocation, constraints []model.Ref) (model.Ref, bool, error)
}

// BaseSodChecker evaluates the contract's constraints against the actor's authorized recorded actions, scoping by subjects.
type BaseSodChecker struct {
	Definitions corpus.AbstractDocumentProvider
	Actions     evidence.Provider
	Evaluator   authority.SodEvaluator
}

var _ SodChecker = (*BaseSodChecker)(nil)

func NewBaseSodChecker(definitions corpus.Source, actions evidence.Provider) *BaseSodChecker {
	return &BaseSodChecker{Definitions: corpus.AbstractDocumentProvider{Source: definitions}, Actions: actions, Evaluator: authority.SodEvaluator{Scopes: authority.ExactSodScopeMatcher{}}}
}

func (c *BaseSodChecker) Conflict(ctx context.Context, inv Invocation, refs []model.Ref) (model.Ref, bool, error) {
	if len(refs) == 0 {
		return model.Ref{}, false, nil
	}
	if c.Actions == nil {
		return model.Ref{}, false, errors.New("separation of duties requires an action source")
	}
	constraints := make([]model.SodConstraint, 0, len(refs))
	for _, ref := range refs {
		sc, err := corpus.ResolveAs[model.SodConstraint](ctx, &c.Definitions, inv.Namespace, ref, model.KindSodConstraint)
		if err != nil {
			return model.Ref{}, false, err
		}
		constraints = append(constraints, sc)
	}
	actions, err := (evidence.Queries{Provider: c.Actions}).ActionsBy(ctx, inv.Namespace, inv.Actor)
	if err != nil {
		return model.Ref{}, false, err
	}
	var performed []authority.SodAction
	for _, a := range actions {
		if authority.Performed(a) {
			performed = append(performed, authority.SodAction{Namespace: a.Namespace, Capability: a.CapabilityID, Scope: authority.SubjectScope(a.Namespace, a.SubjectRefs)})
		}
	}
	proposed := authority.SodAction{Namespace: inv.Namespace, Capability: inv.CapabilityID, Scope: authority.SubjectScope(inv.Namespace, inv.SubjectRefs)}
	return c.Evaluator.Conflict(constraints, performed, proposed)
}
