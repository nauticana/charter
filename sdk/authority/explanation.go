package authority

import (
	"time"

	"github.com/nauticana/charter/sdk/model"
)

const (
	BasisGrant         = "grant"
	BasisDelegation    = "delegation"
	BasisApproval      = "approval"
	BasisPolicy        = "policy"
	BasisSodConstraint = "sod-constraint"

	Satisfied   = "satisfied"
	Unsatisfied = "unsatisfied"
)

// Explanation is why an action would be allowed or denied, taken from the evaluation that would govern it; it
// authorizes nothing (CHR-AUTH-013). It matches schema/agent/authority_explanation.schema.json.
type Explanation struct {
	Actor                 model.ObjectRef `json:"actor"`
	CapabilityID          model.Ref       `json:"capabilityId"`
	ResourceScope         string          `json:"resourceScope,omitempty"`
	OrganizationalContext string          `json:"organizationalContext,omitempty"`
	At                    time.Time       `json:"at"`
	Decision              Result          `json:"decision"`
	Basis                 []Basis         `json:"basis"`
	Missing               []Unmet         `json:"missing,omitempty"`
}

// Basis is one grant, delegation, approval, policy, or constraint the decision considered.
type Basis struct {
	Source   string           `json:"source"`
	Ref      *model.ObjectRef `json:"ref,omitempty"`
	Redacted bool             `json:"redacted,omitempty"`
	Result   string           `json:"result"`
	Reason   string           `json:"reason"`
}

// Unmet is an unmet requirement and, when known, who can resolve it.
type Unmet struct {
	Requirement string            `json:"requirement"`
	Resolvers   []model.ObjectRef `json:"resolvers,omitempty"`
}

// Redact hides every source the viewer may not see, keeping its result and disclosing nothing else of it.
func (x Explanation) Redact(visible func(model.ObjectRef) bool) Explanation {
	basis := make([]Basis, len(x.Basis))
	for i, b := range x.Basis {
		if b.Ref != nil && !visible(*b.Ref) {
			b = Basis{Source: b.Source, Redacted: true, Result: b.Result, Reason: "redacted"}
		}
		basis[i] = b
	}
	x.Basis = basis
	return x
}
