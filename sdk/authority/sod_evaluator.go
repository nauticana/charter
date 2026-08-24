package authority

import (
	"fmt"

	"github.com/nauticana/charter/sdk/model"
)

// SodEvaluator detects separation-of-duties conflicts.
type SodEvaluator struct {
	Scopes SodScopeMatcher
}

type SodScopeMatcher interface {
	SameScope(rule string, performed, proposed SodAction) (bool, error)
}

// ExactSodScopeMatcher treats matching non-empty action scope keys as the same business scope.
type ExactSodScopeMatcher struct{}

var _ SodScopeMatcher = ExactSodScopeMatcher{}

func (ExactSodScopeMatcher) SameScope(_ string, performed, proposed SodAction) (bool, error) {
	return performed.Scope != "" && performed.Scope == proposed.Scope, nil
}

type SodAction struct {
	Namespace  string
	Capability model.Ref
	Scope      string
}

// Conflict returns the constraint violated when one actor performed a conflicting action in the applicable scope.
func (e SodEvaluator) Conflict(constraints []model.SodConstraint, performed []SodAction, proposed SodAction) (model.Ref, bool, error) {
	for _, c := range constraints {
		if !containsCapability(c, proposed) {
			continue
		}
		for _, p := range performed {
			if !sameRef(p.Namespace, p.Capability, proposed.Namespace, proposed.Capability) && containsCapability(c, p) {
				sameScope := true
				if c.Scope != "" {
					if e.Scopes == nil {
						return model.Ref{}, false, fmt.Errorf("constraint %s requires a scope matcher", c.ID)
					}
					var err error
					sameScope, err = e.Scopes.SameScope(c.Scope, p, proposed)
					if err != nil {
						return model.Ref{}, false, err
					}
				}
				if sameScope {
					return model.Ref{Namespace: c.Namespace, ID: c.ID}, true, nil
				}
			}
		}
	}
	return model.Ref{}, false, nil
}

func containsCapability(constraint model.SodConstraint, action SodAction) bool {
	for _, capability := range constraint.ConstrainedActions {
		if sameRef(constraint.Namespace, capability, action.Namespace, action.Capability) {
			return true
		}
	}
	return false
}
