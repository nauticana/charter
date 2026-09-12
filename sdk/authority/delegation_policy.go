package authority

import (
	"fmt"
	"time"

	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/temporal"
)

// DelegationPolicy interprets a grant's delegation block.
type DelegationPolicy struct{}

// RemainingDepth reports how many further delegation hops the grant permits; a direct grant permits none.
func (DelegationPolicy) RemainingDepth(g model.AuthorityGrant) int {
	if g.Delegation == nil {
		return 0
	}
	return g.Delegation.MaxRedelegationDepth
}

// Check fails closed on a malformed, widened, ineffective, or over-limit presented chain; authenticity and revocation
// are the presenting verifier's.
func (DelegationPolicy) Check(chain model.AuthorityChain, at time.Time, measures map[string]Measure) string {
	for index, hop := range chain {
		if hop.GrantRef.ID == "" || hop.Delegator.Kind == "" || hop.Delegator.ID == "" {
			return fmt.Sprintf("delegation hop %d lacks a grant or delegator", index)
		}
		if hop.RemainingDepth < 0 {
			return fmt.Sprintf("delegation hop %d has negative remaining depth", index)
		}
		if hop.Validity == nil || !temporal.NewPeriod(*hop.Validity).Contains(at) {
			return fmt.Sprintf("delegation hop %d is not effective at %s", index, at.Format("2006-01-02"))
		}
		if reason := (LimitEvaluator{}).Check(hop.Limits, measures); reason != "" {
			return fmt.Sprintf("delegation hop %d: %s", index, reason)
		}
		if index+1 < len(chain) && hop.RemainingDepth > chain[index+1].RemainingDepth-1 {
			return fmt.Sprintf("delegation hop %d exceeds its delegator's remaining depth", index)
		}
	}
	return ""
}

// ApprovalRequired reports whether any authority hop makes approval mandatory for the action.
func (DelegationPolicy) ApprovalRequired(chain model.AuthorityChain) bool {
	for _, hop := range chain {
		if hop.ApprovalRequired {
			return true
		}
	}
	return false
}
