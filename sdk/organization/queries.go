package organization

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/temporal"
)

var ErrCycle = errors.New("organization-unit tree contains a cycle")

// Queries derives effective-dated views from any Provider without inferring one relationship from another.
type Queries struct {
	Provider Provider
}

func (q Queries) AssignmentsOf(ctx context.Context, ownerNamespace string, subject model.ObjectRef, at time.Time) ([]model.Assignment, error) {
	want := corpus.ObjectKeyOf(ownerNamespace, subject)
	return q.assignments(ctx, at, func(a model.Assignment) bool {
		return a.Subject.Kind == subject.Kind && corpus.ObjectKeyOf(a.Namespace, a.Subject) == want
	})
}

func (q Queries) AssignmentsTo(ctx context.Context, ownerNamespace string, target model.ObjectRef, at time.Time) ([]model.Assignment, error) {
	want := corpus.ObjectKeyOf(ownerNamespace, target)
	return q.assignments(ctx, at, func(a model.Assignment) bool {
		return a.Target.Kind == target.Kind && corpus.ObjectKeyOf(a.Namespace, a.Target) == want
	})
}

// Occupants lists the human occupancy assignments of a position effective at the given time (CHR-ENT-011).
func (q Queries) Occupants(ctx context.Context, ownerNamespace string, position model.Ref, at time.Time) ([]model.Assignment, error) {
	want := corpus.KeyOf(ownerNamespace, position)
	return q.assignments(ctx, at, func(a model.Assignment) bool {
		return a.Participation == model.ParticipationOccupies && a.Subject.Kind == model.KindHumanIdentity &&
			a.Target.Kind == model.KindPosition && corpus.ObjectKeyOf(a.Namespace, a.Target) == want
	})
}

// Current reports whether an assignment is effective at the given time by validity and lifecycle state (CHR-ENT-005).
func Current(a model.Assignment, at time.Time) bool {
	return temporal.EffectiveAt(a.Validity, at) && (a.LifecycleState == "" || a.LifecycleState == model.LifecycleActive)
}

func (q Queries) assignments(ctx context.Context, at time.Time, keep func(model.Assignment) bool) ([]model.Assignment, error) {
	all, err := q.Provider.Assignments(ctx)
	if err != nil {
		return nil, err
	}
	var out []model.Assignment
	for _, a := range all {
		if keep(a) && Current(a, at) {
			out = append(out, a)
		}
	}
	return out, nil
}

// UnitPath returns the units from the root down to the given unit, failing on a missing parent or a cycle (CHR-ENT-008).
func (q Queries) UnitPath(ctx context.Context, ownerNamespace string, unit model.Ref) ([]model.OrganizationUnit, error) {
	var path []model.OrganizationUnit
	seen := map[corpus.DocumentKey]bool{}
	ref, owner := unit, ownerNamespace
	for {
		u, err := q.Provider.Unit(ctx, owner, ref)
		if err != nil {
			return nil, err
		}
		key := corpus.DocumentKey{Namespace: u.Namespace, ID: u.ID}
		if seen[key] {
			return nil, fmt.Errorf("%w: %s", ErrCycle, key)
		}
		seen[key] = true
		path = append([]model.OrganizationUnit{u}, path...)
		if u.ParentUnitID == nil {
			return path, nil
		}
		ref, owner = *u.ParentUnitID, u.Namespace
	}
}
