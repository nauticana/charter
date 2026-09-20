package validate

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// VerifiedEffectRule checks that postconditions stay within their contract and that no action is recorded as
// executed while a required postcondition lacks a satisfied, observed-fact evaluation (CHR-CAP-009, CHR-CAP-010, CHR-EVID-011).
type VerifiedEffectRule struct{ AbstractRule }

var _ Rule = VerifiedEffectRule{}

func (r VerifiedEffectRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, contract := range allOf[model.CapabilityContract](c, model.KindCapabilityContract) {
		out = append(out, r.declared(contract)...)
	}
	for _, a := range r.actions(c) {
		if contract, ok := resolveAs[model.CapabilityContract](c, a.Namespace, a.CapabilityID, model.KindCapabilityContract); ok {
			out = append(out, r.evaluated(c, a, contract)...)
		}
	}
	return out
}

func (r VerifiedEffectRule) declared(contract model.CapabilityContract) []Finding {
	var out []Finding
	seen := map[string]bool{}
	for _, p := range contract.Postconditions {
		if seen[p.ID] {
			out = append(out, r.finding(contract.Namespace, contract.ID, fmt.Sprintf("postcondition %s is declared more than once", p.ID)))
		}
		seen[p.ID] = true
		for _, outcome := range p.Outcomes {
			if !slices.Contains(contract.Outcomes, outcome) {
				out = append(out, r.finding(contract.Namespace, contract.ID, fmt.Sprintf("postcondition %s applies to outcome %q, which the contract does not declare", p.ID, outcome)))
			}
		}
		if p.ViolationBusinessError != "" && !slices.Contains(contract.BusinessErrors, p.ViolationBusinessError) {
			out = append(out, r.finding(contract.Namespace, contract.ID, fmt.Sprintf("postcondition %s maps a violation to business error %q, which the contract does not declare", p.ID, p.ViolationBusinessError)))
		}
	}
	return out
}

func (r VerifiedEffectRule) evaluated(c *corpus.Corpus, a model.ActionRecord, contract model.CapabilityContract) []Finding {
	var out []Finding
	results := map[string]string{}
	for _, e := range a.PostconditionEvaluations {
		if !slices.ContainsFunc(contract.Postconditions, func(p model.Postcondition) bool { return p.ID == e.PostconditionID }) {
			out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("evaluates postcondition %s, which %s does not declare", e.PostconditionID, contract.ID)))
			continue
		}
		results[e.PostconditionID] = e.Result
		if e.Result == model.PostconditionUnknown {
			continue
		}
		for _, ref := range e.EvidenceRecordIDs {
			if rec, ok := resolveAs[model.EvidenceRecord](c, a.Namespace, ref, model.KindEvidenceRecord); ok && rec.Category != model.CategoryObservedFact {
				out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("postcondition %s rests on %s, a %s record rather than an observed fact", e.PostconditionID, rec.ID, rec.Category)))
			}
		}
	}
	if a.Disposition != model.DispositionExecuted {
		return out
	}
	for _, p := range contract.RequiredPostconditions(a.Outcome) {
		if results[p.ID] != model.PostconditionSatisfied {
			out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("executed while required postcondition %s is %s", p.ID, cmp.Or(results[p.ID], "not evaluated"))))
		}
	}
	return out
}

// ReconciliationRule checks that a reconciling action resolves an unknown attempt of the same actor and capability,
// and follows it in time (CHR-EVID-004, CHR-EVID-012).
type ReconciliationRule struct{ AbstractRule }

var _ Rule = ReconciliationRule{}

func (r ReconciliationRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, a := range r.actions(c) {
		if a.ReconcilesActionID == nil {
			continue
		}
		prior, ok := resolveAs[model.ActionRecord](c, a.Namespace, *a.ReconcilesActionID, model.KindActionRecord)
		if !ok {
			continue
		}
		switch {
		case prior.Disposition != model.DispositionUnknown:
			out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("reconciles %s, whose disposition is %s rather than unknown", prior.ID, prior.Disposition)))
		case prior.CapabilityID != a.CapabilityID || prior.Actor != a.Actor:
			out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("reconciles %s, an attempt of another actor or capability", prior.ID)))
		case a.ActionTime.Before(prior.ActionTime):
			out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("reconciles %s but precedes it", prior.ID)))
		}
	}
	return out
}
