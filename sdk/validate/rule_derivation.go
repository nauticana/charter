package validate

import (
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// DerivationRule checks that a document derives from a document of its own kind other than itself, or from a source
// declared external (CHR-CONF-015).
type DerivationRule struct {
	AbstractRule
	Meta *SchemaMeta
}

var _ Rule = DerivationRule{}

func (r DerivationRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, d := range c.Documents() {
		p := d.DerivedFrom
		switch {
		case p == nil:
		case !r.Meta.Kinds[string(p.Source.Kind)]:
			if !p.Source.External {
				out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("derivation source kind %s is not a Charter kind and is not declared external", p.Source.Kind)))
			}
		case p.Source.Kind != d.Kind:
			out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("derives from %s %s, not a %s", p.Source.Kind, p.Source.ID, d.Kind)))
		case corpus.ObjectKeyOf(d.Namespace, p.Source) == (corpus.DocumentKey{Namespace: d.Namespace, ID: d.ID}):
			out = append(out, r.finding(d.Namespace, d.ID, "derives from itself"))
		}
	}
	return out
}

// published reports whether a document is past drafting, after which its derivation is fixed (CHR-CONF-015).
func published(d *model.Document) bool {
	return d.LifecycleState != model.LifecycleProposed && d.LifecycleState != model.LifecyclePlanned
}
