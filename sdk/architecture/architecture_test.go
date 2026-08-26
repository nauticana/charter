package architecture

import (
	"context"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func TestHarborArchitecture(t *testing.T) {
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ns := "harbor.example"
	p := NewBaseProvider(c)
	chk := Checker{Provider: p}
	gap, err := p.Gap(ctx, ns, model.Ref{ID: "GAP-ORDER-EXCEPTION-EVIDENCE"})
	if err != nil {
		t.Fatal(err)
	}
	if issues, err := chk.CheckGap(ctx, gap); err != nil || len(issues) != 0 {
		t.Errorf("harbor gap: %v %v", issues, err)
	}
	gap.BaselineStateID, gap.TargetStateID = gap.TargetStateID, gap.BaselineStateID
	issues, err := chk.CheckGap(ctx, gap)
	if err != nil || len(issues) != 3 {
		t.Errorf("swapped gap: want baseline-type, target-type, and ordering issues, got %v %v", issues, err)
	}
	item, err := p.RoadmapItem(ctx, ns, model.Ref{ID: "ROADMAP-GOVERNED-EXCEPTION-PILOT"})
	if err != nil {
		t.Fatal(err)
	}
	if issues, err := chk.CheckRoadmapItem(ctx, item); err != nil || len(issues) != 0 {
		t.Errorf("harbor roadmap item: %v %v", issues, err)
	}
	item.AddressesGapIDs = append(item.AddressesGapIDs, model.Ref{ID: "GAP-NOWHERE"})
	if issues, _ := chk.CheckRoadmapItem(ctx, item); len(issues) != 1 || issues[0].Requirement != "CHR-ARCH-005" {
		t.Errorf("dangling gap not reported: %v", issues)
	}
	ent := model.Ref{ID: "ENT-HARBOR"}
	if states, err := chk.StatesAt(ctx, ns, ent, model.StateBaseline, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)); err != nil || len(states) != 1 || states[0].ID != "ARCH-BASELINE-2026-Q1" {
		t.Errorf("baseline at Q1: %v %v", states, err)
	}
	if states, err := chk.StatesAt(ctx, ns, ent, model.StateBaseline, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)); err != nil || len(states) != 0 {
		t.Errorf("baseline after Q1: %v %v", states, err)
	}
	if states, err := chk.StatesAt(ctx, ns, ent, model.StateTarget, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)); err != nil || len(states) != 1 {
		t.Errorf("target in Q4: %v %v", states, err)
	}
	target := model.ArchitectureState{StateType: model.StateTarget}
	target.ID = "T"
	if issues := chk.CheckState(target); len(issues) != 2 {
		t.Errorf("unbounded target without authority: %v", issues)
	}
}
