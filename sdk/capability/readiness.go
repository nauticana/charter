package capability

import (
	"context"
	"slices"

	"github.com/nauticana/charter/sdk/binding"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/information"
	"github.com/nauticana/charter/sdk/model"
)

// Blocker is one reason a capability cannot yet be invoked, with the requirement the invoker would deny it under.
type Blocker struct {
	Requirement string
	Reason      string
}

// Readiness reports, before a binding goes live, every denial that follows from the documents and the composition
// rather than from an action: the contract, the binding and its declared features, the input and output schemas and
// the data bindings of their shapes, the authority binding that maps the contract's required authority, and the
// approval gate, separation-of-duties checker, ledger, effect observer, predicate evaluator, and escalator the contract requires. No blockers means only action-specific gates remain.
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
	bind, err := lookup.CapabilityBindingFor(ctx, namespace, capabilityID, profile)
	if err != nil {
		out = append(out, Blocker{"CHR-BIND-009", "binding: " + err.Error()})
	} else if err := binding.Features(bind.FeatureSupport).Require(features...); err != nil {
		out = append(out, Blocker{"CHR-BIND-006", "binding " + bind.ID + ": " + err.Error()})
	}
	for _, ds := range []*model.DataSchema{contract.InputSchema, contract.OutputSchema} {
		if ds == nil {
			continue
		}
		if _, err := i.dataSchema(ctx, namespace, ds); err != nil {
			out = append(out, Blocker{"CHR-CAP-011", "data schema: " + err.Error()})
		} else if ds.InformationID != nil && bind.ID != "" {
			out = append(out, i.unmapped(ctx, namespace, *ds.InformationID, bind)...)
		}
	}
	switch declared, pre := predicates(contract); {
	case declared && i.Predicates == nil:
		out = append(out, Blocker{"CHR-CAP-012", "capability declares predicates but no predicate evaluator is composed"})
	case pre && i.Observer == nil:
		out = append(out, Blocker{"CHR-CAP-012", "capability declares predicate preconditions but no observer is composed"})
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
		if i.Escalation == nil && slices.ContainsFunc(contract.Outcomes, func(o string) bool { return len(contract.RequiredPostconditions(o)) > 1 }) {
			out = append(out, Blocker{"CHR-EVID-016", "an outcome with several required postconditions can be partial, which needs an escalator"})
		}
	}
	return out
}

// unmapped reports the shape properties the data bindings of the capability binding's system profile leave unmapped
// or map to paths that do not resolve (CHR-INFO-010).
func (i *BaseInvoker) unmapped(ctx context.Context, namespace string, info model.Ref, bind model.CapabilityBinding) []Blocker {
	def, err := i.Definitions.Definition(ctx, namespace, info)
	if err != nil {
		return []Blocker{{"CHR-INFO-010", "information " + info.ID + ": " + err.Error()}}
	}
	all, err := i.Bindings.DataBindings(ctx)
	if err != nil {
		return []Blocker{{"CHR-INFO-010", "data bindings: " + err.Error()}}
	}
	var out []Blocker
	for _, db := range all {
		if corpus.KeyOf(db.Namespace, db.SystemProfileID) != corpus.KeyOf(bind.Namespace, bind.SystemProfileID) ||
			corpus.KeyOf(db.Namespace, db.InformationDefinitionID) != (corpus.DocumentKey{Namespace: def.Namespace, ID: def.ID}) {
			continue
		}
		for _, problem := range information.CheckMappings(ctx, i.Definitions, def, db.FieldMappings) {
			out = append(out, Blocker{"CHR-INFO-010", "data binding " + db.ID + ": " + problem})
		}
	}
	return out
}
