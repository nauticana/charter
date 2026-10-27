package validate

import (
	"context"
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/information"
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

// ShapeRule checks each information shape and that every data binding to a shaped definition resolves its paths and
// maps every required property (CHR-INFO-010).
type ShapeRule struct{ AbstractRule }

var _ Rule = ShapeRule{}

func (r ShapeRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, def := range allOf[model.InformationDefinition](c, model.KindInformationDefinition) {
		if def.Shape != nil {
			for _, problem := range information.CheckShape(*def.Shape) {
				out = append(out, r.finding(def.Namespace, def.ID, problem))
			}
		}
	}
	defs := information.NewBaseProvider(c)
	for _, b := range allOf[model.DataBinding](c, model.KindDataBinding) {
		def, ok := resolveAs[model.InformationDefinition](c, b.Namespace, b.InformationDefinitionID, model.KindInformationDefinition)
		if !ok {
			continue
		}
		for _, problem := range information.CheckMappings(context.Background(), defs, def, b.FieldMappings) {
			out = append(out, r.finding(b.Namespace, b.ID, problem))
		}
	}
	return out
}
