package validate

import (
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/temporal"
)

// ExternalIdentifiersRule checks that a document's external identifiers for one system profile are ordered and never
// effective at the same time (CHR-ENT-013).
type ExternalIdentifiersRule struct{ AbstractRule }

var _ Rule = ExternalIdentifiersRule{}

func (r ExternalIdentifiersRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, d := range c.Documents() {
		ids := d.ExternalIdentifiers
		for i, a := range ids {
			if a.Validity != nil && !temporal.NewPeriod(*a.Validity).Ordered() {
				out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("external identifier %s ends before it starts", a.ExternalID)))
			}
			for _, b := range ids[i+1:] {
				if corpus.KeyOf(d.Namespace, a.SystemProfileID) == corpus.KeyOf(d.Namespace, b.SystemProfileID) && concurrent(a.Validity, b.Validity) {
					out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("external identifiers %s and %s in %s are effective at the same time", a.ExternalID, b.ExternalID, a.SystemProfileID.ID)))
				}
			}
		}
	}
	return out
}

// concurrent reports whether two optional validities share a day; an absent validity is open-ended.
func concurrent(a, b *model.Validity) bool {
	return a == nil || b == nil || temporal.NewPeriod(*a).Overlaps(temporal.NewPeriod(*b))
}
