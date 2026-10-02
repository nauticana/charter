// Package evidencetest is the contract suite every evidence.Store and evidence.Provider implementation runs from its
// own tests.
package evidencetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/model"
)

// Run checks append-only storage through an AbstractSink and the Provider lookups over the same store. newStore
// returns an empty store; newProvider reads it back.
func Run(t *testing.T, newStore func(t *testing.T) evidence.Store, newProvider func(evidence.Store) evidence.Provider) {
	ctx := context.Background()
	at := time.Date(2026, 6, 18, 17, 0, 0, 0, time.UTC)
	actions := []model.ActionRecord{
		Action("a.example", "ACT-2", "EXEC-1", at.Add(2*time.Minute)),
		Action("a.example", "ACT-1", "EXEC-1", at),
		Action("a.example", "ACT-3", "EXEC-2", at.Add(time.Minute)),
		Action("b.example", "ACT-1", "EXEC-1", at.Add(time.Minute)),
	}
	seeded := func(t *testing.T) evidence.Store {
		t.Helper()
		s := newStore(t)
		sink := &evidence.AbstractSink{Store: s, Digester: evidence.BaseSHA256Digester{}}
		for _, a := range actions {
			if err := sink.Action(ctx, a); err != nil {
				t.Fatalf("append %s:%s: %v", a.Namespace, a.ID, err)
			}
		}
		return s
	}

	t.Run("the store is append-only and keyed by namespace and id", func(t *testing.T) {
		s := seeded(t)
		sink := &evidence.AbstractSink{Store: s, Digester: evidence.BaseSHA256Digester{}}
		if err := sink.Action(ctx, actions[0]); !errors.Is(err, evidence.ErrDuplicate) {
			t.Errorf("duplicate append: %v", err)
		}
		d, err := s.Fetch(ctx, corpus.DocumentKey{Namespace: "b.example", ID: "ACT-1"})
		if err != nil || d.Kind != model.KindActionRecord || d.Namespace != "b.example" {
			t.Errorf("fetch: %+v %v", d, err)
		}
		if _, err := s.Fetch(ctx, corpus.DocumentKey{Namespace: "a.example", ID: "ACT-MISSING"}); !errors.Is(err, corpus.ErrNotFound) {
			t.Errorf("missing fetch: %v", err)
		}
		if all, err := s.List(ctx, model.KindActionRecord); err != nil || len(all) != len(actions) {
			t.Errorf("list: %d %v", len(all), err)
		}
		if none, err := s.List(ctx, model.KindEscalation); err != nil || len(none) != 0 {
			t.Errorf("list of an absent kind: %d %v", len(none), err)
		}
	})

	t.Run("the provider resolves and lists the stored actions", func(t *testing.T) {
		p := newProvider(seeded(t))
		a, err := p.Action(ctx, "a.example", model.Ref{ID: "ACT-3"})
		if err != nil || a.RuntimeContext.ExecutionContextID != "EXEC-2" {
			t.Errorf("action: %+v %v", a, err)
		}
		if all, err := p.Actions(ctx); err != nil || len(all) != len(actions) {
			t.Errorf("actions: %d %v", len(all), err)
		}
	})

	t.Run("actions of one execution context stay within its namespace, in time order", func(t *testing.T) {
		q := evidence.Queries{Provider: newProvider(seeded(t))}
		in, err := q.ActionsIn(ctx, "a.example", "EXEC-1")
		if err != nil || len(in) != 2 || in[0].ID != "ACT-1" || in[1].ID != "ACT-2" || in[0].Namespace != "a.example" {
			t.Errorf("actions in EXEC-1: %+v %v", in, err)
		}
		if none, err := q.ActionsIn(ctx, "a.example", "EXEC-MISSING"); err != nil || len(none) != 0 {
			t.Errorf("unknown context: %+v %v", none, err)
		}
		if _, err := q.ActionsIn(ctx, "", "EXEC-1"); !errors.Is(err, evidence.ErrNoExecutionContext) {
			t.Errorf("blank namespace: %v", err)
		}
	})
}

// Action is a minimal valid action record in one execution context.
func Action(namespace, id, executionContextID string, at time.Time) model.ActionRecord {
	a := model.ActionRecord{Actor: model.ObjectRef{Kind: model.KindAgentIdentity, ID: "AGENT-1"}, CapabilityID: model.Ref{ID: "CAP-1"},
		RuntimeContext: model.RuntimeContext{RuntimeInstanceID: model.Ref{ID: "RT-1"}, ExecutionContextID: executionContextID},
		ActionTime:     at, Outcome: "done", Disposition: model.DispositionExecuted, OperationClass: model.OperationRead}
	a.CharterSpecVersion, a.Namespace, a.ID, a.Kind = "1.2.0", namespace, id, model.KindActionRecord
	return a
}
