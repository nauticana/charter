package validate

import (
	"fmt"
	"slices"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// WorkConceptRule checks that concepts of one classification carry distinct external identifiers and that a concept's
// broader and related concepts belong to its classification and are not itself (CHR-ENT-014).
type WorkConceptRule struct{ AbstractRule }

var _ Rule = WorkConceptRule{}

func (r WorkConceptRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	seen := map[[2]string]string{}
	for _, w := range allOf[model.WorkConcept](c, model.KindWorkConcept) {
		classification := corpus.KeyOf(w.Namespace, w.ClassificationID)
		key := [2]string{classification.String(), w.ExternalID}
		if other, dup := seen[key]; dup {
			out = append(out, r.finding(w.Namespace, w.ID, fmt.Sprintf("external identifier %q is also carried by %s in %s", w.ExternalID, other, classification.ID)))
		}
		seen[key] = w.ID
		self := corpus.DocumentKey{Namespace: w.Namespace, ID: w.ID}
		for _, ref := range slices.Concat(w.BroaderConceptIDs, w.RelatedConceptIDs) {
			target, ok := resolveAs[model.WorkConcept](c, w.Namespace, ref, model.KindWorkConcept)
			switch {
			case !ok:
			case corpus.KeyOf(w.Namespace, ref) == self:
				out = append(out, r.finding(w.Namespace, w.ID, "names itself as a broader or related concept"))
			case corpus.KeyOf(target.Namespace, target.ClassificationID) != classification:
				out = append(out, r.finding(w.Namespace, w.ID, fmt.Sprintf("concept %s belongs to %s, not %s", target.ID, target.ClassificationID.ID, classification.ID)))
			}
		}
	}
	return out
}
