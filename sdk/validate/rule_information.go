package validate

import (
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

type PolicyScopeRule struct{ AbstractRule }

var _ Rule = PolicyScopeRule{}

func (r PolicyScopeRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, def := range allOf[model.InformationDefinition](c, model.KindInformationDefinition) {
		key := corpus.DocumentKey{Namespace: def.Namespace, ID: def.ID}
		for _, ref := range def.GovernancePolicyIDs {
			policy, ok := resolveAs[model.InformationGovernancePolicy](c, def.Namespace, ref, model.KindInformationGovernancePolicy)
			if !ok {
				continue
			}
			covered := false
			for _, s := range policy.Scope {
				if s.Kind == model.KindInformationDefinition && corpus.ObjectKeyOf(policy.Namespace, s) == key {
					covered = true
				}
			}
			if !covered {
				out = append(out, r.finding(def.Namespace, def.ID, fmt.Sprintf("policy %s does not list %s in its scope", policy.ID, def.ID)))
			}
		}
	}
	return out
}
