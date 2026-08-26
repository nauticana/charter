package architecture

import (
	"context"
	"fmt"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/temporal"
)

// Issue is one consistency failure linked to the requirement it violates.
type Issue struct {
	Requirement string
	Message     string
}

// Checker verifies version/effective-context and reference consistency of architecture documents.
type Checker struct {
	Provider Provider
}

// CheckGap verifies that a gap compares a time-bounded baseline state with a target state (CHR-ARCH-004).
func (c Checker) CheckGap(ctx context.Context, gap model.Gap) ([]Issue, error) {
	var issues []Issue
	baseline, err := c.Provider.State(ctx, gap.Namespace, gap.BaselineStateID)
	if err != nil {
		issues = append(issues, Issue{"CHR-ARCH-004", "baseline: " + err.Error()})
	} else {
		if baseline.StateType != model.StateBaseline {
			issues = append(issues, Issue{"CHR-ARCH-004", fmt.Sprintf("baselineStateId %s is a %s state", baseline.ID, baseline.StateType)})
		}
		issues = append(issues, c.CheckState(baseline)...)
	}
	target, err := c.Provider.State(ctx, gap.Namespace, gap.TargetStateID)
	if err != nil {
		issues = append(issues, Issue{"CHR-ARCH-004", "target: " + err.Error()})
	} else {
		if target.StateType != model.StateTarget {
			issues = append(issues, Issue{"CHR-ARCH-004", fmt.Sprintf("targetStateId %s is a %s state", target.ID, target.StateType)})
		}
		issues = append(issues, c.CheckState(target)...)
	}
	if baseline.Validity != nil && target.Validity != nil && string(target.Validity.From) < string(baseline.Validity.From) {
		issues = append(issues, Issue{"CHR-ARCH-006", fmt.Sprintf("target %s starts before baseline %s", target.ID, baseline.ID)})
	}
	return issues, nil
}

// CheckState verifies that a state is time-bounded and that a target state names its decision authority (CHR-ARCH-002, CHR-ARCH-003, CHR-ARCH-006).
func (c Checker) CheckState(s model.ArchitectureState) []Issue {
	var issues []Issue
	switch {
	case s.Validity == nil:
		issues = append(issues, Issue{"CHR-ARCH-006", fmt.Sprintf("state %s is not time-bounded", s.ID)})
	case !temporal.NewPeriod(*s.Validity).Ordered():
		issues = append(issues, Issue{"CHR-ARCH-006", fmt.Sprintf("state %s validity is reversed", s.ID)})
	}
	if s.StateType == model.StateTarget && s.DecisionAuthority == nil {
		issues = append(issues, Issue{"CHR-ARCH-003", fmt.Sprintf("target state %s names no decision authority", s.ID)})
	}
	if s.StateType == model.StateBaseline && s.Description == "" {
		issues = append(issues, Issue{"CHR-ARCH-002", fmt.Sprintf("baseline state %s describes no observed conditions", s.ID)})
	}
	return issues
}

// CheckRoadmapItem verifies that the item addresses resolvable gaps and declares an owner (CHR-ARCH-005).
func (c Checker) CheckRoadmapItem(ctx context.Context, item model.RoadmapItem) ([]Issue, error) {
	var issues []Issue
	if len(item.AddressesGapIDs) == 0 {
		issues = append(issues, Issue{"CHR-ARCH-005", fmt.Sprintf("roadmap item %s addresses no gap", item.ID)})
	}
	for _, ref := range item.AddressesGapIDs {
		if _, err := c.Provider.Gap(ctx, item.Namespace, ref); err != nil {
			issues = append(issues, Issue{"CHR-ARCH-005", fmt.Sprintf("roadmap item %s: %v", item.ID, err)})
		}
	}
	if item.Owner.ID == "" {
		issues = append(issues, Issue{"CHR-ARCH-005", fmt.Sprintf("roadmap item %s has no owner", item.ID)})
	}
	if item.Status == "" {
		issues = append(issues, Issue{"CHR-ARCH-005", fmt.Sprintf("roadmap item %s has no status", item.ID)})
	}
	return issues, nil
}

// StatesAt lists an enterprise's states of one type effective at a time, so later analysis reads the states of its day (CHR-ARCH-006).
func (c Checker) StatesAt(ctx context.Context, ownerNamespace string, enterprise model.Ref, stateType string, at time.Time) ([]model.ArchitectureState, error) {
	all, err := c.Provider.States(ctx)
	if err != nil {
		return nil, err
	}
	want := corpus.KeyOf(ownerNamespace, enterprise)
	var out []model.ArchitectureState
	for _, s := range all {
		if s.StateType == stateType && s.EnterpriseID != nil && corpus.KeyOf(s.Namespace, *s.EnterpriseID) == want && temporal.EffectiveAt(s.Validity, at) {
			out = append(out, s)
		}
	}
	return out, nil
}
