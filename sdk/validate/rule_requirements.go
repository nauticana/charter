package validate

import (
	"fmt"
	"slices"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/model"
)

// EvidenceRequirementsRule checks that every structured evidence requirement names an attribute of its information's
// shape, which the information must declare, with an age no wider than the information's bound, and that an executed action cites a current fact, within
// that age, for every required item of its contract (CHR-CAP-013).
type EvidenceRequirementsRule struct{ AbstractRule }

var _ Rule = EvidenceRequirementsRule{}

func (r EvidenceRequirementsRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, t := range allOf[model.Task](c, model.KindTask) {
		out = append(out, r.declared(c, t.Namespace, t.ID, t.EvidenceRequirements)...)
	}
	for _, cc := range allOf[model.CapabilityContract](c, model.KindCapabilityContract) {
		out = append(out, r.declared(c, cc.Namespace, cc.ID, cc.EvidenceRequirements)...)
	}
	for _, a := range r.actions(c) {
		contract, ok := resolveAs[model.CapabilityContract](c, a.Namespace, a.CapabilityID, model.KindCapabilityContract)
		if !ok || a.Disposition != model.DispositionExecuted {
			continue
		}
		cited := citedFacts(c, a)
		for _, e := range model.RequiredEvidence(contract.EvidenceRequirements) {
			def, ok := resolveAs[model.InformationDefinition](c, contract.Namespace, *e.InformationID, model.KindInformationDefinition)
			if !ok {
				continue
			}
			k := slices.IndexFunc(cited, func(rec model.EvidenceRecord) bool {
				return rec.Fact.Attribute == e.Attribute && corpus.KeyOf(rec.Namespace, rec.Fact.InformationID) == (corpus.DocumentKey{Namespace: def.Namespace, ID: def.ID})
			})
			if k < 0 {
				out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("cites no fact for required evidence %s.%s of %s", def.ID, e.Attribute, contract.ID)))
			} else if problem := standing(c, cited[k], evidence.Narrowed(def, e.MaxAgeSeconds), a.ActionTime); problem != "" {
				out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("required evidence %s.%s: %s", def.ID, e.Attribute, problem)))
			}
		}
	}
	return out
}

func (r EvidenceRequirementsRule) declared(c *corpus.Corpus, namespace, id string, reqs []model.EvidenceRequirement) []Finding {
	var out []Finding
	for _, e := range reqs {
		if e.InformationID == nil {
			continue
		}
		def, ok := resolveAs[model.InformationDefinition](c, namespace, *e.InformationID, model.KindInformationDefinition)
		if !ok {
			continue
		}
		switch {
		case def.Shape == nil:
			out = append(out, r.finding(namespace, id, fmt.Sprintf("evidence requirement names %s.%s, but %s declares no shape", def.ID, e.Attribute, def.ID)))
		case !slices.ContainsFunc(def.Shape.Properties, func(p model.ShapeProperty) bool { return p.Name == e.Attribute }):
			out = append(out, r.finding(namespace, id, fmt.Sprintf("evidence requirement names %s.%s, which its shape does not declare", def.ID, e.Attribute)))
		}
		if bound := def.FactAuthority; bound != nil && bound.MaxAgeSeconds > 0 && e.MaxAgeSeconds > bound.MaxAgeSeconds {
			out = append(out, r.finding(namespace, id, fmt.Sprintf("evidence requirement %s.%s allows %d seconds, wider than the %d %s declares", def.ID, e.Attribute, e.MaxAgeSeconds, bound.MaxAgeSeconds, def.ID)))
		}
	}
	return out
}
