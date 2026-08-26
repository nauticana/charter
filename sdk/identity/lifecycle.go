// Package identity resolves human and agent identities and establishes their lifecycle state at a point in time.
package identity

import (
	"errors"
	"fmt"
	"time"

	"github.com/nauticana/charter/sdk/model"
)

var (
	ErrNotIdentity = errors.New("reference is not a human or agent identity")
	ErrInactive    = errors.New("identity is not active")
	ErrLifecycle   = errors.New("lifecycle state cannot be established")
	ErrNoProvider  = errors.New("no identity provider")
)

// Lifecycle is an identity's current state plus its retained, effective-dated transitions.
type Lifecycle struct {
	Current string
	History []model.LifecycleTransition
}

// StateAt establishes the state at t from strictly ordered transitions consistent with the current state (CHR-ID-009).
func (l Lifecycle) StateAt(t time.Time) (string, error) {
	if len(l.History) == 0 {
		return "", fmt.Errorf("%w: lifecycleHistory is missing", ErrLifecycle)
	}
	state := ""
	var previous time.Time
	for _, transition := range l.History {
		if transition.EffectiveAt.IsZero() || (!previous.IsZero() && !transition.EffectiveAt.After(previous)) {
			return "", fmt.Errorf("%w: lifecycleHistory is not strictly ordered", ErrLifecycle)
		}
		previous = transition.EffectiveAt
		if !transition.EffectiveAt.After(t) {
			state = transition.State
		}
	}
	if l.Current != l.History[len(l.History)-1].State {
		return "", fmt.Errorf("%w: current lifecycleState differs from latest transition", ErrLifecycle)
	}
	if state == "" {
		return "", fmt.Errorf("%w: no transition is effective at %s", ErrLifecycle, t.Format(time.RFC3339))
	}
	return state, nil
}

// ActiveAt fails closed unless the identity was active at t (CHR-ID-005).
func (l Lifecycle) ActiveAt(t time.Time) error {
	state, err := l.StateAt(t)
	if err != nil {
		return err
	}
	if state != model.LifecycleActive {
		return fmt.Errorf("%w: %s at %s", ErrInactive, state, t.Format(time.RFC3339))
	}
	return nil
}
