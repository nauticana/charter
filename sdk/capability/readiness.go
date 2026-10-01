package capability

import (
	"context"

	"github.com/nauticana/charter/sdk/binding"
	"github.com/nauticana/charter/sdk/model"
)

// Blocker is one reason a capability cannot yet be invoked, with the requirement the invoker would deny it under.
type Blocker struct {
	Requirement string
	Reason      string
}

// Readiness reports, before a binding goes live, every denial that follows from the documents and the composition
// rather than from an action: the contract, the binding and its declared features, the authority binding that maps
// the contract's required authority, and the approval gate, separation-of-duties checker, ledger, and effect observer
// the contract requires. No blockers means only action-specific gates remain.
func (i *BaseInvoker) Readiness(ctx context.Context, namespace string, capabilityID model.Ref, profile *model.Ref, features ...string) []Blocker {
	if i.Catalog == nil || i.Identities == nil || i.Authority == nil || i.Bindings == nil || i.Transport == nil || i.Evidence == nil || i.IDs == nil {
		return []Blocker{{"CHR-AUTH-010", "invoker is not fully composed"}}
	}
	contract, err := i.Catalog.Contract(ctx, namespace, capabilityID)
	if err != nil {
		return []Blocker{{"CHR-CAP-001", "capability contract: " + err.Error()}}
	}
	var out []Blocker
	lookup := binding.Lookup{Provider: i.Bindings}
	if bind, err := lookup.CapabilityBindingFor(ctx, namespace, capabilityID, profile); err != nil {
		out = append(out, Blocker{"CHR-BIND-009", "binding: " + err.Error()})
	} else if err := binding.Features(bind.FeatureSupport).Require(features...); err != nil {
		out = append(out, Blocker{"CHR-BIND-006", "binding " + bind.ID + ": " + err.Error()})
	}
	if _, err := lookup.AuthorityBindingFor(ctx, namespace, capabilityID, profile); err != nil {
		out = append(out, Blocker{"CHR-BIND-006", "authority " + contract.RequiredAuthority + " is not mapped: " + err.Error()})
	}
	if contract.Constraints.ApprovalRequired && i.Approvals == nil {
		out = append(out, Blocker{"CHR-AUTH-009", "approval required but no approval gate is composed"})
	}
	if len(contract.Constraints.SodConstraintIDs) > 0 && i.Sod == nil {
		out = append(out, Blocker{"CHR-AUTH-007", "separation of duties declared but no checker is composed"})
	}
	if contract.Idempotency.Mutating {
		if i.Ledger == nil {
			out = append(out, Blocker{"CHR-SEC-008", "mutating capability requires an idempotency ledger"})
		}
		if len(contract.RequiredPostconditions("")) > 0 && i.Observer == nil {
			out = append(out, Blocker{"CHR-CAP-010", "capability declares required postconditions but no effect observer is composed"})
		}
	}
	return out
}
