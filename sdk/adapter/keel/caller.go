package keel

import (
	"context"
	"fmt"
	"time"

	"github.com/nauticana/charter/sdk/identity"
)

// Caller resolves the Charter identity acting behind a keel-authenticated request.
type Caller struct {
	Identities identity.Resolver
	Map        IdentityMap
}

// Actor returns the mapped identity, established active at the action time (CHR-SEC-001, CHR-ID-005).
func (c Caller) Actor(ctx context.Context, at time.Time) (identity.Actor, Session, error) {
	if c.Map == nil {
		return identity.Actor{}, Session{}, fmt.Errorf("%w: no identity map", ErrUnmapped)
	}
	if c.Identities == nil {
		return identity.Actor{}, Session{}, identity.ErrNoProvider
	}
	s, err := SessionFromContext(ctx)
	if err != nil {
		return identity.Actor{}, Session{}, err
	}
	ref, err := c.Map.Actor(ctx, s)
	if err != nil {
		return identity.Actor{}, s, err
	}
	actor, err := c.Identities.ActiveAt(ctx, ref.Namespace, ref, at)
	if err != nil {
		return identity.Actor{}, s, err
	}
	return actor, s, nil
}
