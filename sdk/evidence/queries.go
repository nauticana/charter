package evidence

import (
	"context"
	"errors"
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

var ErrSupersessionCycle = errors.New("supersession chain is cyclic")

// Queries derives provenance and attribution views from any Provider.
type Queries struct {
	Provider Provider
}

// ActionsBy lists the recorded actions of one actor (CHR-AUTH-007 needs them for separation of duties).
func (q Queries) ActionsBy(ctx context.Context, ownerNamespace string, actor model.ObjectRef) ([]model.ActionRecord, error) {
	all, err := q.Provider.Actions(ctx)
	if err != nil {
		return nil, err
	}
	want := corpus.ObjectKeyOf(ownerNamespace, actor)
	var out []model.ActionRecord
	for _, a := range all {
		if a.Actor.Kind == actor.Kind && corpus.ObjectKeyOf(a.Namespace, a.Actor) == want {
			out = append(out, a)
		}
	}
	return out, nil
}

// Lineage follows supersession from a record back to the original, newest first (CHR-EVID-004, CHR-INFO-004).
func (q Queries) Lineage(ctx context.Context, ownerNamespace string, record model.Ref) ([]model.EvidenceRecord, error) {
	var out []model.EvidenceRecord
	seen := map[corpus.DocumentKey]bool{}
	ref, owner := record, ownerNamespace
	for {
		r, err := q.Provider.Record(ctx, owner, ref)
		if err != nil {
			return nil, err
		}
		key := corpus.DocumentKey{Namespace: r.Namespace, ID: r.ID}
		if seen[key] {
			return nil, fmt.Errorf("%w: %s", ErrSupersessionCycle, key)
		}
		seen[key] = true
		out = append(out, r)
		if r.Supersedes == nil {
			return out, nil
		}
		ref, owner = *r.Supersedes, r.Namespace
	}
}

// SupersededBy lists the corrections that supersede a record.
func (q Queries) SupersededBy(ctx context.Context, ownerNamespace string, record model.Ref) ([]model.EvidenceRecord, error) {
	all, err := q.Provider.Records(ctx)
	if err != nil {
		return nil, err
	}
	want := corpus.KeyOf(ownerNamespace, record)
	var out []model.EvidenceRecord
	for _, r := range all {
		if r.Supersedes != nil && corpus.KeyOf(r.Namespace, *r.Supersedes) == want {
			out = append(out, r)
		}
	}
	return out, nil
}
