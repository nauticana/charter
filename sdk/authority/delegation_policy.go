package authority

import "github.com/nauticana/charter/sdk/model"

// DelegationPolicy interprets a grant's delegation block.
type DelegationPolicy struct{}

// RemainingDepth reports how many further delegation hops the grant permits; a direct grant permits none.
func (DelegationPolicy) RemainingDepth(g model.AuthorityGrant) int {
	if g.Delegation == nil {
		return 0
	}
	return g.Delegation.MaxRedelegationDepth
}
