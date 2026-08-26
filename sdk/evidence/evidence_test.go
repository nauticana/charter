package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func record(id, category, content string, supersedes *model.Ref) model.EvidenceRecord {
	r := model.EvidenceRecord{Category: category, Content: content, RecordedAt: time.Date(2026, 6, 18, 17, 3, 12, 0, time.UTC), Supersedes: supersedes}
	r.CharterSpecVersion, r.Namespace, r.ID = "draft", "t.example", id
	r.EnterpriseID = &model.Ref{ID: "ENT"}
	return r
}

func TestSinkIsAppendOnlyWithSupersessionAndIntegrity(t *testing.T) {
	ctx := context.Background()
	sink := NewBaseMemorySink()
	observed := record("EVR-1", model.CategoryObservedFact, "40 units", nil)
	if err := sink.Record(ctx, observed); err != nil {
		t.Fatal(err)
	}
	if err := sink.Record(ctx, observed); !errors.Is(err, ErrDuplicate) {
		t.Errorf("duplicate accepted: %v", err)
	}
	if err := sink.Record(ctx, record("EVR-X", model.CategoryObservedFact, "x", &model.Ref{ID: "EVR-NOWHERE"})); !errors.Is(err, ErrSupersedes) {
		t.Errorf("supersession of missing record accepted: %v", err)
	}
	if err := sink.Record(ctx, record("EVR-2", model.CategoryObservedFact, "38 units", &model.Ref{ID: "EVR-1"})); err != nil {
		t.Fatal(err)
	}
	provider := NewBaseProvider(sink.Store)
	if lineage, err := (Queries{provider}).Lineage(ctx, "t.example", model.Ref{ID: "EVR-2"}); err != nil || len(lineage) != 2 || lineage[1].ID != "EVR-1" {
		t.Errorf("lineage: %v %v", lineage, err)
	}
	if by, err := (Queries{provider}).SupersededBy(ctx, "t.example", model.Ref{ID: "EVR-1"}); err != nil || len(by) != 1 || by[0].ID != "EVR-2" {
		t.Errorf("superseded by: %v %v", by, err)
	}
	if original, err := provider.Record(ctx, "t.example", model.Ref{ID: "EVR-1"}); err != nil || original.Content != "40 units" {
		t.Errorf("superseded record must stay addressable: %+v %v", original, err)
	}

	bundle := model.EvidenceBundle{Subject: model.ObjectRef{Kind: model.KindProcessInstance, ID: "PI-1"}, RecordIDs: []model.Ref{{ID: "EVR-1"}, {ID: "EVR-2"}}, AssuranceProfile: "standard"}
	bundle.CharterSpecVersion, bundle.Namespace, bundle.ID = "draft", "t.example", "EVID-1"
	bundle.EnterpriseID = &model.Ref{ID: "ENT"}
	missing := bundle
	missing.ID, missing.RecordIDs = "EVID-MISSING", []model.Ref{{ID: "EVR-NOWHERE"}}
	if err := sink.Bundle(ctx, missing); !errors.Is(err, ErrBundleRecord) {
		t.Errorf("bundle over missing record accepted: %v", err)
	}
	if err := sink.Bundle(ctx, bundle); err != nil {
		t.Fatal(err)
	}
	stored, err := provider.Bundle(ctx, "t.example", model.Ref{ID: "EVID-1"})
	if err != nil || stored.Integrity.Method != "sha256-hash-chain" || stored.Integrity.Value == "" {
		t.Fatalf("stored bundle: %+v %v", stored.Integrity, err)
	}
	verifier := Verifier{Source: sink.Store, Digester: BaseSHA256Digester{}}
	if err := verifier.Verify(ctx, "t.example", model.Ref{ID: "EVID-1"}); err != nil {
		t.Errorf("verify: %v", err)
	}
	tampered := bundle
	tampered.ID, tampered.Integrity = "EVID-2", model.Integrity{Method: "sha256-hash-chain", Value: "sha256:0000"}
	if err := sink.Bundle(ctx, tampered); !errors.Is(err, ErrIntegrity) {
		t.Errorf("wrong integrity value accepted: %v", err)
	}
	if err := (&AbstractSink{}).Record(ctx, observed); !errors.Is(err, ErrNoStore) {
		t.Errorf("sink without store: %v", err)
	}
}

func TestHarborQueries(t *testing.T) {
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	q := Queries{Provider: NewBaseProvider(c)}
	ctx := context.Background()
	if actions, err := q.ActionsBy(ctx, "harbor.example", model.ObjectRef{Kind: model.KindAgentIdentity, ID: "AGENT-ORDER-EXCEPTION-COORDINATOR"}); err != nil || len(actions) != 2 {
		t.Errorf("actions by agent: %d %v", len(actions), err)
	}
	if lineage, err := q.Lineage(ctx, "harbor.example", model.Ref{ID: "EVR-0042-STOCK-CORRECTED"}); err != nil || len(lineage) != 2 || lineage[1].ID != "EVR-0042-STOCK-OBSERVED" {
		t.Errorf("harbor lineage: %v %v", lineage, err)
	}
}
