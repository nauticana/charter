package validate

import (
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
)

// SpecVersion reports documents that target a specification version other than the one being validated against (CHR-CONF-001).
type SpecVersion struct {
	Version string
}

var _ Validator = SpecVersion{}

func (v SpecVersion) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, d := range c.Documents() {
		if d.CharterSpecVersion != v.Version {
			out = append(out, Finding{RuleID: "spec-version", Requirements: []string{"CHR-CONF-001"}, Class: ClassStructural, Namespace: d.Namespace, DocumentID: d.ID,
				Path: "/charterSpecVersion", Message: fmt.Sprintf("targets specification %q, validated against %q", d.CharterSpecVersion, v.Version)})
		}
	}
	return out
}
