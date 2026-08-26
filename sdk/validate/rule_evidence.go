package validate

import (
	"context"
	"errors"

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
