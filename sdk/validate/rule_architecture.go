package validate

import (
	"context"

	"github.com/nauticana/charter/sdk/architecture"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

type GapStatesRule struct{ AbstractRule }

var _ Rule = GapStatesRule{}

func (r GapStatesRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	checker := architecture.Checker{Provider: architecture.NewBaseProvider(c)}
	for _, gap := range allOf[model.Gap](c, model.KindGap) {
		issues, err := checker.CheckGap(context.Background(), gap)
		if err != nil {
			out = append(out, r.finding(gap.Namespace, gap.ID, err.Error()))
			continue
		}
		for _, issue := range issues {
			out = append(out, r.finding(gap.Namespace, gap.ID, issue.Requirement+": "+issue.Message))
		}
	}
	return out
}
