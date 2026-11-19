package authority

import (
	"fmt"
	"time"

	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/temporal"
)

// Held reports why an actor does not hold a grant at t: outside its validity or revoked (CHR-AUTH-005). An empty
// reason means the grant is held, though Lapse may still suspend it.
func Held(g model.AuthorityGrant, at time.Time) string {
	if g.Validity == nil || !temporal.NewPeriod(*g.Validity).Contains(at) {
		return "not effective at " + at.Format("2006-01-02")
	}
	if g.RevokedAt != nil && !at.Before(*g.RevokedAt) {
		return "revoked at " + g.RevokedAt.Format(time.RFC3339)
	}
	return ""
}

// Lapse reports why a grant is not in force at t: not held (Held), or suspended because its recertification is
// overdue or was made by its holder (CHR-AUTH-011). An empty reason means the grant is in force.
func Lapse(g model.AuthorityGrant, at time.Time) string {
	if reason := Held(g, at); reason != "" {
		return reason
	}
	c := g.Certification
	if c == nil {
		return ""
	}
	if sameObjectRef(g.Namespace, c.CertifiedBy, g.Namespace, g.Actor) {
		return "certified by its own holder"
	}
	if due := c.CertifiedAt.AddDate(0, 0, c.IntervalDays); c.IntervalDays < 1 || at.After(due) {
		return "certification overdue since " + due.Format(time.RFC3339)
	}
	return ""
}

// Excuse removes from constraints the one an exception relieves actor of at t (CHR-AUTH-012). An exception that is
// self-granted, for another actor, ineffective, revoked, or for a constraint not in the list fails closed.
func Excuse(constraints []model.SodConstraint, e model.SodException, namespace string, actor model.ObjectRef, at time.Time) ([]model.SodConstraint, error) {
	switch {
	case !sameObjectRef(e.Namespace, e.Beneficiary, namespace, actor):
		return nil, fmt.Errorf("exception %s relieves %s, not %s", e.ID, e.Beneficiary.ID, actor.ID)
	case sameObjectRef(e.Namespace, e.Grantor, e.Namespace, e.Beneficiary):
		return nil, fmt.Errorf("exception %s is self-granted", e.ID)
	case e.Validity == nil || !temporal.NewPeriod(*e.Validity).Contains(at):
		return nil, fmt.Errorf("exception %s is not effective at %s", e.ID, at.Format("2006-01-02"))
	case e.RevokedAt != nil && !at.Before(*e.RevokedAt):
		return nil, fmt.Errorf("exception %s was revoked at %s", e.ID, e.RevokedAt.Format(time.RFC3339))
	}
	for i, c := range constraints {
		if sameRef(e.Namespace, e.SodConstraintID, c.Namespace, model.Ref{ID: c.ID}) {
			return append(constraints[:i:i], constraints[i+1:]...), nil
		}
	}
	return nil, fmt.Errorf("exception %s relieves %s, which does not govern this action", e.ID, e.SodConstraintID.ID)
}
