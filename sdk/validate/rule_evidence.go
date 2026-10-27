package validate

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/model"
)

type SupersessionRule struct{ AbstractRule }

var _ Rule = SupersessionRule{}

func (r SupersessionRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	queries := evidence.Queries{Provider: evidence.NewBaseProvider(c)}
	for _, rec := range allOf[model.EvidenceRecord](c, model.KindEvidenceRecord) {
		if rec.Supersedes == nil {
			continue
		}
		if _, err := queries.Lineage(context.Background(), rec.Namespace, model.Ref{ID: rec.ID}); errors.Is(err, evidence.ErrSupersessionCycle) {
			out = append(out, r.finding(rec.Namespace, rec.ID, err.Error()))
		}
	}
	return out
}

// RecordedOutcomeRule checks that an executed action reports an outcome its capability declares and a business-error
// action a declared business error (CHR-CAP-001, CHR-EVID-010).
type RecordedOutcomeRule struct{ AbstractRule }

var _ Rule = RecordedOutcomeRule{}

func (r RecordedOutcomeRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, a := range r.actions(c) {
		contract, ok := resolveAs[model.CapabilityContract](c, a.Namespace, a.CapabilityID, model.KindCapabilityContract)
		if !ok {
			continue
		}
		switch a.Disposition {
		case model.DispositionExecuted:
			if !slices.Contains(contract.Outcomes, a.Outcome) {
				out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("executed with outcome %q, which %s does not declare (%v)", a.Outcome, contract.ID, contract.Outcomes)))
			}
		case model.DispositionBusinessError:
			if !slices.Contains(contract.BusinessErrors, a.Outcome) {
				out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("business error %q is not declared by %s (%v)", a.Outcome, contract.ID, contract.BusinessErrors)))
			}
		}
	}
	return out
}

// CurrentFactsRule checks that an executed action relies on no fact that was stale, superseded, outranked, or in
// conflict at its action time (CHR-EVID-014, CHR-EVID-015).
type CurrentFactsRule struct{ AbstractRule }

var _ Rule = CurrentFactsRule{}

func (r CurrentFactsRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	provider := evidence.NewBaseProvider(c)
	q := evidence.Queries{Provider: provider}
	for _, a := range r.actions(c) {
		if a.Disposition != model.DispositionExecuted {
			continue
		}
		cited := slices.Clone(a.EvidenceRecordIDs)
		for _, e := range a.PreconditionEvaluations {
			cited = append(cited, e.EvidenceRecordIDs...)
		}
		for _, e := range a.PostconditionEvaluations {
			cited = append(cited, e.EvidenceRecordIDs...)
		}
		seen := map[corpus.DocumentKey]bool{}
		for _, ref := range cited {
			key := corpus.KeyOf(a.Namespace, ref)
			rec, err := provider.Record(context.Background(), a.Namespace, ref)
			if seen[key] || err != nil || rec.Fact == nil {
				continue
			}
			seen[key] = true
			def, ok := resolveAs[model.InformationDefinition](c, rec.Namespace, rec.Fact.InformationID, model.KindInformationDefinition)
			if !ok {
				continue
			}
			states, err := q.CurrentFacts(context.Background(), rec.Namespace, rec.Fact.Subject, def, a.ActionTime)
			if err != nil {
				out = append(out, r.finding(a.Namespace, a.ID, err.Error()))
				continue
			}
			if k := slices.IndexFunc(states, func(s evidence.FactState) bool { return s.Record.ID == rec.ID && s.Record.Namespace == rec.Namespace }); k < 0 {
				out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("relies on fact %s, which was not yet recorded or valid", rec.ID)))
			} else if states[k].State != evidence.FactCurrent {
				out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("relies on fact %s, which was %s: %s", rec.ID, states[k].State, states[k].Rule)))
			}
		}
	}
	return out
}
