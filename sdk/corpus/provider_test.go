package corpus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/model"
)

func harbor(t *testing.T) *AbstractDocumentProvider {
	c, err := NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	return &AbstractDocumentProvider{Source: c}
}

func TestProviderResolvesByKindAndFailsClosed(t *testing.T) {
	p := harbor(t)
	ctx := context.Background()
	pos, err := ResolveAs[model.Position](ctx, p, "harbor.example", model.Ref{ID: "POS-CREDIT-MANAGER"}, model.KindPosition)
	if err != nil || pos.OrganizationUnitID.ID != "OU-CREDIT-CONTROL" {
		t.Fatalf("position: %+v %v", pos, err)
	}
	if _, err := p.Resolve(ctx, "harbor.example", model.Ref{ID: "POS-CREDIT-MANAGER"}, model.KindOrganizationUnit); !errors.Is(err, ErrKindMismatch) {
		t.Errorf("kind mismatch not reported: %v", err)
	}
	if _, err := p.Resolve(ctx, "harbor.example", model.Ref{ID: "POS-NOWHERE"}, model.KindPosition); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing document not reported: %v", err)
	}
	if _, err := p.ResolveObject(ctx, "harbor.example", model.ObjectRef{Kind: "SalesOrder", ID: "ORD-1", External: true}); !errors.Is(err, ErrExternal) {
		t.Errorf("external reference resolved: %v", err)
	}
	if _, err := (&AbstractDocumentProvider{}).Resolve(ctx, "x", model.Ref{ID: "y"}, ""); !errors.Is(err, ErrNoSource) {
		t.Errorf("missing source not reported: %v", err)
	}
}

func TestEffectiveAtUsesValidity(t *testing.T) {
	p := harbor(t)
	assignments, err := EffectiveAs[model.Assignment](context.Background(), p, model.KindAssignment, time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range assignments {
		if a.ID == "ASGN-PRIYA-PP-PLN-01" {
			t.Error("assignment that ended in 2025 reported effective in 2026")
		}
	}
	if len(assignments) == 0 {
		t.Error("no effective assignments")
	}
}
