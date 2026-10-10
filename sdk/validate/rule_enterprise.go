package validate

import (
	"fmt"
	"slices"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/temporal"
)

type UnitTreeRule struct{ AbstractRule }
type PositionPlacementRule struct{ AbstractRule }
type ValidityOrderedRule struct{ AbstractRule }
type HumanOccupancyRule struct{ AbstractRule }
type ApplicabilityRule struct{ AbstractRule }

var (
	_ Rule = UnitTreeRule{}
	_ Rule = PositionPlacementRule{}
	_ Rule = ValidityOrderedRule{}
	_ Rule = HumanOccupancyRule{}
	_ Rule = ApplicabilityRule{}
)

func (r UnitTreeRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	byEnt := map[corpus.DocumentKey][]model.OrganizationUnit{}
	for _, d := range c.OfKind(model.KindOrganizationUnit) {
		u, err := corpus.Decode[model.OrganizationUnit](d)
		if err != nil || u.EnterpriseID == nil {
			continue
		}
		entNamespace := u.EnterpriseID.Namespace
		if entNamespace == "" {
			entNamespace = u.Namespace
		}
		ent := corpus.DocumentKey{Namespace: entNamespace, ID: u.EnterpriseID.ID}
		byEnt[ent] = append(byEnt[ent], u)
	}
	for ent, us := range byEnt {
		roots := 0
		for _, u := range us {
			if u.ParentUnitID == nil {
				roots++
			}
		}
		if roots != 1 {
			out = append(out, r.finding(ent.Namespace, ent.ID, fmt.Sprintf("%d root units", roots)))
		}
		for _, u := range us {
			seen := map[corpus.DocumentKey]bool{}
			cur := u
			for cur.ParentUnitID != nil {
				curKey := corpus.DocumentKey{Namespace: cur.Namespace, ID: cur.ID}
				if seen[curKey] {
					out = append(out, r.finding(u.Namespace, u.ID, "cycle in organization-unit tree"))
					break
				}
				seen[curKey] = true
				nextDoc, ok := c.Resolve(cur.Namespace, *cur.ParentUnitID)
				if !ok || nextDoc.Kind != model.KindOrganizationUnit {
					out = append(out, r.finding(u.Namespace, u.ID, fmt.Sprintf("parent %s is not a unit of %s", cur.ParentUnitID.ID, ent.ID)))
					break
				}
				next, err := corpus.Decode[model.OrganizationUnit](nextDoc)
				if err != nil || next.EnterpriseID == nil {
					out = append(out, r.finding(u.Namespace, u.ID, fmt.Sprintf("parent %s is invalid", cur.ParentUnitID.ID)))
					break
				}
				nextEntNamespace := next.EnterpriseID.Namespace
				if nextEntNamespace == "" {
					nextEntNamespace = next.Namespace
				}
				if (corpus.DocumentKey{Namespace: nextEntNamespace, ID: next.EnterpriseID.ID}) != ent {
					out = append(out, r.finding(u.Namespace, u.ID, fmt.Sprintf("parent %s is not a unit of %s", cur.ParentUnitID.ID, ent.ID)))
					break
				}
				cur = next
			}
		}
	}
	return out
}

func (r PositionPlacementRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, d := range c.OfKind(model.KindPosition) {
		p, err := corpus.Decode[model.Position](d)
		if err == nil && c.KindOfRef(p.Namespace, p.OrganizationUnitID) != model.KindOrganizationUnit {
			out = append(out, r.finding(d.Namespace, d.ID, "organizationUnitId is not an OrganizationUnit"))
		}
	}
	for _, d := range c.OfKind(model.KindOrganizationUnit) {
		u, err := corpus.Decode[model.OrganizationUnit](d)
		if err == nil && u.ParentUnitID != nil && c.KindOfRef(u.Namespace, *u.ParentUnitID) == model.KindPosition {
			out = append(out, r.finding(d.Namespace, d.ID, "parentUnitId is a Position"))
		}
	}
	return out
}

func (r ValidityOrderedRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, d := range c.Documents() {
		if d.Validity != nil && !temporal.NewPeriod(*d.Validity).Ordered() {
			out = append(out, r.finding(d.Namespace, d.ID, "validity.to before validity.from"))
		}
	}
	return out
}

func (r HumanOccupancyRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, d := range c.OfKind(model.KindAssignment) {
		a, err := corpus.Decode[model.Assignment](d)
		if err != nil || a.Participation != model.ParticipationOccupies {
			continue
		}
		if subject, ok := c.ResolveObject(a.Namespace, a.Subject); !ok || subject.Kind != model.KindHumanIdentity {
			out = append(out, r.finding(d.Namespace, d.ID, "occupies subject is not a HumanIdentity document"))
		}
		if target, ok := c.ResolveObject(a.Namespace, a.Target); !ok || target.Kind != model.KindPosition {
			out = append(out, r.finding(d.Namespace, d.ID, "occupies target is not a Position document"))
		}
	}
	return out
}

// Validate finds two assignments of one target and participation, at least one declaring applicability, that may both
// apply to the same work at the same time without exactly one of them being the default (CHR-ENT-012).
func (r ApplicabilityRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	all := allOf[model.Assignment](c, model.KindAssignment)
	for i, a := range all {
		for _, b := range all[i+1:] {
			if a.Applicability == nil && b.Applicability == nil || a.Participation != b.Participation || a.Target.Kind != b.Target.Kind ||
				corpus.ObjectKeyOf(a.Namespace, a.Target) != corpus.ObjectKeyOf(b.Namespace, b.Target) ||
				a.Validity == nil || b.Validity == nil || !temporal.NewPeriod(*a.Validity).Overlaps(temporal.NewPeriod(*b.Validity)) ||
				!applicabilityOverlaps(a.Applicability, b.Applicability) || isDefault(a) != isDefault(b) {
				continue
			}
			out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("applies to the same work as %s and exactly one must be the default", b.ID)))
		}
	}
	return out
}

func isDefault(a model.Assignment) bool { return a.Applicability != nil && a.Applicability.Default }

// applicabilityOverlaps is false only when a dimension both constrain has no value in common; an absent applicability matches everything.
func applicabilityOverlaps(a, b *model.Applicability) bool {
	if a == nil || b == nil {
		return true
	}
	for _, x := range a.Criteria {
		for _, y := range b.Criteria {
			if x.Dimension == y.Dimension && !slices.ContainsFunc(x.Values, func(v string) bool { return slices.Contains(y.Values, v) }) {
				return false
			}
		}
	}
	return true
}
