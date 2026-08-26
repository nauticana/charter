package identity

import (
	"context"
	"fmt"
	"time"

	"github.com/nauticana/charter/sdk/model"
)

// AbstractResolver resolves actors through any Provider; embed it and supply the Provider.
type AbstractResolver struct {
	Provider Provider
}

var _ Resolver = (*AbstractResolver)(nil)

func (r *AbstractResolver) Resolve(ctx context.Context, ownerNamespace string, ref model.ObjectRef) (Actor, error) {
	if r.Provider == nil {
		return Actor{}, ErrNoProvider
	}
	id := model.Ref{Namespace: ref.Namespace, ID: ref.ID}
	switch ref.Kind {
	case model.KindHumanIdentity:
		h, err := r.Provider.Human(ctx, ownerNamespace, id)
		if err != nil {
			return Actor{}, err
		}
		return Actor{Envelope: h.Envelope, Lifecycle: Lifecycle{Current: h.LifecycleState, History: h.LifecycleHistory}}, nil
	case model.KindAgentIdentity:
		a, err := r.Provider.Agent(ctx, ownerNamespace, id)
		if err != nil {
			return Actor{}, err
		}
		return Actor{Envelope: a.Envelope, Lifecycle: Lifecycle{Current: a.LifecycleState, History: a.LifecycleHistory}}, nil
	}
	return Actor{}, fmt.Errorf("%w: %s %s", ErrNotIdentity, ref.Kind, ref.ID)
}

func (r *AbstractResolver) ActiveAt(ctx context.Context, ownerNamespace string, ref model.ObjectRef, at time.Time) (Actor, error) {
	actor, err := r.Resolve(ctx, ownerNamespace, ref)
	if err != nil {
		return Actor{}, err
	}
	if err := actor.Lifecycle.ActiveAt(at); err != nil {
		return Actor{}, fmt.Errorf("%s: %w", actor.ID, err)
	}
	return actor, nil
}
