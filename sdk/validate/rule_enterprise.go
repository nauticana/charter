package validate

import (
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/temporal"
)

type UnitTreeRule struct{ AbstractRule }
type PositionPlacementRule struct{ AbstractRule }
type ValidityOrderedRule struct{ AbstractRule }
type HumanOccupancyRule struct{ AbstractRule }

var (
	_ Rule = UnitTreeRule{}
	_ Rule = PositionPlacementRule{}
	_ Rule = ValidityOrderedRule{}
	_ Rule = HumanOccupancyRule{}
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
