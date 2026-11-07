package validate

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/temporal"
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
	for _, a := range r.actions(c) {
		if a.Disposition != model.DispositionExecuted {
			continue
		}
		for _, rec := range citedFacts(c, a) {
			def, ok := resolveAs[model.InformationDefinition](c, rec.Namespace, rec.Fact.InformationID, model.KindInformationDefinition)
			if !ok {
				continue
			}
			if problem := standing(c, rec, def, a.ActionTime); problem != "" {
				out = append(out, r.finding(a.Namespace, a.ID, problem))
			}
		}
	}
	return out
}

// citedFacts returns, once each, the fact-stating records an action cites in its evidence and condition evaluations.
func citedFacts(c *corpus.Corpus, a model.ActionRecord) []model.EvidenceRecord {
	cited := slices.Clone(a.EvidenceRecordIDs)
	for _, e := range a.PreconditionEvaluations {
		cited = append(cited, e.EvidenceRecordIDs...)
	}
	for _, e := range a.PostconditionEvaluations {
		cited = append(cited, e.EvidenceRecordIDs...)
	}
	seen := map[corpus.DocumentKey]bool{}
	var out []model.EvidenceRecord
	for _, ref := range cited {
		key := corpus.KeyOf(a.Namespace, ref)
		rec, ok := resolveAs[model.EvidenceRecord](c, a.Namespace, ref, model.KindEvidenceRecord)
		if ok && rec.Fact != nil && !seen[key] {
			seen[key] = true
			out = append(out, rec)
		}
	}
	return out
}

// standing explains why a cited fact could not be relied on at a decision time under def, or is empty when it was current.
func standing(c *corpus.Corpus, rec model.EvidenceRecord, def model.InformationDefinition, at time.Time) string {
	states, err := (evidence.Queries{Provider: evidence.NewBaseProvider(c)}).CurrentFacts(context.Background(), rec.Namespace, rec.Fact.Subject, def, at)
	if err != nil {
		return err.Error()
	}
	k := slices.IndexFunc(states, func(s evidence.FactState) bool { return s.Record.ID == rec.ID && s.Record.Namespace == rec.Namespace })
	switch {
	case k < 0:
		return fmt.Sprintf("relies on fact %s, which was not yet recorded or valid", rec.ID)
	case states[k].State != evidence.FactCurrent:
		return fmt.Sprintf("relies on fact %s, which was %s: %s", rec.ID, states[k].State, states[k].Rule)
	}
	return ""
}

// MeasurementRule checks that a measurement cites only executed actions whose effect was not partial, absent, or
// unobserved, states an integer value for a currency measure, and agrees with any baseline citing it (CHR-PROC-011).
type MeasurementRule struct{ AbstractRule }

var _ Rule = MeasurementRule{}

func (r MeasurementRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	records := map[corpus.DocumentKey]model.EvidenceRecord{}
	for _, rec := range allOf[model.EvidenceRecord](c, model.KindEvidenceRecord) {
		records[corpus.DocumentKey{Namespace: rec.Namespace, ID: rec.ID}] = rec
		m := rec.Measurement
		if m == nil {
			continue
		}
		if !temporal.NewPeriod(m.Window).Ordered() {
			out = append(out, r.finding(rec.Namespace, rec.ID, "measurement window ends before it starts"))
		}
		if measure, ok := resolveAs[model.OutcomeMeasure](c, rec.Namespace, m.MeasureID, model.KindOutcomeMeasure); ok && measure.Currency != "" {
			if _, err := m.Value.Int64(); err != nil {
				out = append(out, r.finding(rec.Namespace, rec.ID, fmt.Sprintf("measurement of %s states %s, not integer minor units of %s", measure.ID, m.Value, measure.Currency)))
			}
		}
		for _, ref := range m.ActionRecordIDs {
			a, ok := resolveAs[model.ActionRecord](c, rec.Namespace, ref, model.KindActionRecord)
			switch {
			case !ok:
			case a.Disposition != model.DispositionExecuted:
				out = append(out, r.finding(rec.Namespace, rec.ID, fmt.Sprintf("measurement rests on %s, whose disposition is %s", a.ID, a.Disposition)))
			case a.Effect != "" && a.Effect != model.EffectComplete:
				out = append(out, r.finding(rec.Namespace, rec.ID, fmt.Sprintf("measurement rests on %s, whose effect is %s", a.ID, a.Effect)))
			}
		}
	}
	for _, om := range allOf[model.OutcomeMeasure](c, model.KindOutcomeMeasure) {
		if om.Baseline == nil {
			continue
		}
		for _, ref := range om.Baseline.EvidenceRecordIDs {
			rec, ok := records[corpus.KeyOf(om.Namespace, ref)]
			m := rec.Measurement
			if !ok || m == nil {
				continue
			}
			if corpus.KeyOf(rec.Namespace, m.MeasureID) != (corpus.DocumentKey{Namespace: om.Namespace, ID: om.ID}) || m.Value != om.Baseline.Value || m.Window != om.Baseline.Window {
				out = append(out, r.finding(om.Namespace, om.ID, fmt.Sprintf("baseline disagrees with the measurement %s states", rec.ID)))
			}
		}
	}
	return out
}
