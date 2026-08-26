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
