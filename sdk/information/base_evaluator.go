package information

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/temporal"
)

var (
	ErrNoPolicy    = errors.New("no governance policy applies")
	ErrScope       = errors.New("governance policy does not cover the information")
	ErrConflict    = errors.New("conflicting governance rules")
	ErrNoRetention = errors.New("no retention rule for the artifact kind")
)

// BaseEvaluator decides governance questions from the policies an information definition declares.
type BaseEvaluator struct {
	Provider Provider
}

var _ Evaluator = (*BaseEvaluator)(nil)

func (e *BaseEvaluator) Use(ctx context.Context, req UseRequest) Decision {
	_, policies, err := e.policies(ctx, req.Namespace, req.Information, req.At)
	if err != nil {
		return decisionFor(err)
	}
	refs := refsOf(policies)
	var permit, deny []model.InformationGovernancePolicy
	for _, p := range policies {
		if slices.Contains(p.PermittedUses, req.Purpose) {
			permit = append(permit, p)
		} else {
			deny = append(deny, p)
		}
	}
	switch {
	case len(deny) == 0:
		return Decision{Result: Allowed, Reason: fmt.Sprintf("purpose %q permitted by every applicable policy", req.Purpose), Requirement: "CHR-INFO-007", Policies: refs}
	case len(permit) == 0:
		return Decision{Result: Denied, Reason: fmt.Sprintf("purpose %q is not a permitted use", req.Purpose), Requirement: "CHR-INFO-007", Policies: refs}
	}
	winner, err := precedence(append(permit, deny...))
	if err != nil {
		return Decision{Result: Denied, Reason: err.Error(), Requirement: "CHR-INFO-008", Policies: refs}
	}
	if slices.Contains(winner.PermittedUses, req.Purpose) {
		return Decision{Result: Allowed, Reason: fmt.Sprintf("policy %s takes precedence and permits %q", winner.ID, req.Purpose), Requirement: "CHR-INFO-008", Policies: refs}
	}
	return Decision{Result: Denied, Reason: fmt.Sprintf("policy %s takes precedence and does not permit %q", winner.ID, req.Purpose), Requirement: "CHR-INFO-008", Policies: refs}
}

// Retention returns the rule for an artifact kind; differing rules are resolved by declared precedence or fail closed (CHR-INFO-005, CHR-INFO-008).
func (e *BaseEvaluator) Retention(ctx context.Context, owner string, information model.Ref, artifactKind string, at time.Time) (model.RetentionRule, error) {
	_, policies, err := e.policies(ctx, owner, information, at)
	if err != nil {
		return model.RetentionRule{}, err
	}
	rules := map[corpus.DocumentKey]model.RetentionRule{}
	var holders []model.InformationGovernancePolicy
	for _, p := range policies {
		for _, r := range p.RetentionAndDeletion {
			if r.ArtifactKind == artifactKind {
				rules[corpus.DocumentKey{Namespace: p.Namespace, ID: p.ID}] = r
				holders = append(holders, p)
			}
		}
	}
	if len(holders) == 0 {
		return model.RetentionRule{}, fmt.Errorf("%w: %s", ErrNoRetention, artifactKind)
	}
	first := rules[corpus.DocumentKey{Namespace: holders[0].Namespace, ID: holders[0].ID}]
	same := true
	for _, r := range rules {
		if r != first {
			same = false
		}
	}
	if same {
		return first, nil
	}
	winner, err := precedence(holders)
	if err != nil {
		return model.RetentionRule{}, err
	}
	return rules[corpus.DocumentKey{Namespace: winner.Namespace, ID: winner.ID}], nil
}

// AccessConstraints unites the constraints of every applicable policy (CHR-INFO-002).
func (e *BaseEvaluator) AccessConstraints(ctx context.Context, owner string, information model.Ref, at time.Time) ([]string, error) {
	_, policies, err := e.policies(ctx, owner, information, at)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, p := range policies {
		for _, c := range p.AccessConstraints {
			if !slices.Contains(out, c) {
				out = append(out, c)
			}
		}
	}
	return out, nil
}

// policies resolves the definition's policies effective at the time; each must place the definition in scope.
func (e *BaseEvaluator) policies(ctx context.Context, owner string, information model.Ref, at time.Time) (model.InformationDefinition, []model.InformationGovernancePolicy, error) {
	def, err := e.Provider.Definition(ctx, owner, information)
	if err != nil {
		return def, nil, err
	}
	key := corpus.DocumentKey{Namespace: def.Namespace, ID: def.ID}
	var out []model.InformationGovernancePolicy
	for _, ref := range def.GovernancePolicyIDs {
		p, err := e.Provider.Policy(ctx, def.Namespace, ref)
		if err != nil {
			return def, nil, err
		}
		if !temporal.EffectiveAt(p.Validity, at) || p.LifecycleState == model.LifecycleRetired {
			continue
		}
		if !covers(p, key) {
			return def, nil, fmt.Errorf("%w: %s does not list %s", ErrScope, p.ID, def.ID)
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return def, nil, fmt.Errorf("%w: %s at %s", ErrNoPolicy, def.ID, at.Format("2006-01-02"))
	}
	return def, out, nil
}

func covers(p model.InformationGovernancePolicy, key corpus.DocumentKey) bool {
	for _, s := range p.Scope {
		if s.Kind == model.KindInformationDefinition && corpus.ObjectKeyOf(p.Namespace, s) == key {
			return true
		}
	}
	return false
}

// precedence picks the highest-precedence policy when every conflicting policy opts into declared precedence; ties and fail-closed policies deny.
func precedence(conflicting []model.InformationGovernancePolicy) (model.InformationGovernancePolicy, error) {
	var winner model.InformationGovernancePolicy
	tie := false
	for i, p := range conflicting {
		if p.ConflictBehavior != model.ConflictPrecedence {
			return winner, fmt.Errorf("%w: policy %s fails closed", ErrConflict, p.ID)
		}
		switch {
		case i == 0 || p.Precedence > winner.Precedence:
			winner, tie = p, false
		case p.Precedence == winner.Precedence:
			tie = true
		}
	}
	if tie {
		return winner, fmt.Errorf("%w: equal precedence %d", ErrConflict, winner.Precedence)
	}
	return winner, nil
}

func decisionFor(err error) Decision {
	if errors.Is(err, ErrNoPolicy) || errors.Is(err, ErrScope) {
		return Decision{Result: Denied, Reason: err.Error(), Requirement: "CHR-INFO-002"}
	}
	return Decision{Result: Error, Reason: err.Error(), Requirement: "CHR-INFO-008"}
}

func refsOf(policies []model.InformationGovernancePolicy) []model.Ref {
	out := make([]model.Ref, 0, len(policies))
	for _, p := range policies {
		out = append(out, model.Ref{Namespace: p.Namespace, ID: p.ID})
	}
	return out
}
