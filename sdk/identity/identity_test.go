package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func TestResolveHarborActors(t *testing.T) {
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	r := NewBaseResolver(c)
	ctx := context.Background()
	at := time.Date(2026, 6, 18, 17, 12, 30, 0, time.UTC)
	agent := model.ObjectRef{Kind: model.KindAgentIdentity, ID: "AGENT-ORDER-EXCEPTION-COORDINATOR"}
	actor, err := r.ActiveAt(ctx, "harbor.example", agent, at)
	if err != nil || actor.Ref().Namespace != "harbor.example" || actor.Ref().Kind != model.KindAgentIdentity {
		t.Fatalf("agent: %+v %v", actor.Ref(), err)
	}
	if _, err := r.ActiveAt(ctx, "harbor.example", agent, time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)); !errors.Is(err, ErrLifecycle) {
		t.Errorf("agent before activation: %v", err)
	}
	human := model.ObjectRef{Kind: model.KindHumanIdentity, ID: "HUMAN-ALEX-RIVERA"}
	if actor, err := r.Resolve(ctx, "harbor.example", human); err != nil || actor.Lifecycle.Current != model.LifecycleActive {
		t.Errorf("human: %+v %v", actor, err)
	}
	if _, err := r.ActiveAt(ctx, "harbor.example", human, at); !errors.Is(err, ErrLifecycle) {
		t.Errorf("human without history must not be established active: %v", err)
	}
	if _, err := r.Resolve(ctx, "harbor.example", model.ObjectRef{Kind: model.KindPosition, ID: "POS-CREDIT-MANAGER"}); !errors.Is(err, ErrNotIdentity) {
		t.Errorf("position resolved as identity: %v", err)
	}
	if _, err := r.Resolve(ctx, "harbor.example", model.ObjectRef{Kind: model.KindHumanIdentity, ID: agent.ID}); !errors.Is(err, corpus.ErrKindMismatch) {
		t.Errorf("declared kind not enforced: %v", err)
	}
}

func TestLifecycleStateAt(t *testing.T) {
	l := Lifecycle{Current: model.LifecycleSuspended, History: []model.LifecycleTransition{
		{State: model.LifecycleActive, EffectiveAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{State: model.LifecycleSuspended, EffectiveAt: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
	}}
	if err := l.ActiveAt(time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Error(err)
	}
	if err := l.ActiveAt(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)); !errors.Is(err, ErrInactive) {
		t.Errorf("suspended identity accepted: %v", err)
	}
	l.Current = model.LifecycleActive
	if _, err := l.StateAt(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)); !errors.Is(err, ErrLifecycle) {
		t.Errorf("inconsistent current state accepted: %v", err)
	}
}
