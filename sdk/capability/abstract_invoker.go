package capability

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/nauticana/charter/sdk/authority"
	"github.com/nauticana/charter/sdk/binding"
	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/identity"
	"github.com/nauticana/charter/sdk/information"
	"github.com/nauticana/charter/sdk/model"
)

// AbstractInvoker runs the governed invocation pipeline around an abstract binding.Executor: contract and version,
// actor lifecycle, authority, approval, separation of duties, information governance, binding feature support,
// idempotency, execution, outcome verification, evidence, and escalation of stopped actions. Every gate fails closed
// and every governed decision is recorded (CHR-AUTH-010, CHR-AGENT-008).
type AbstractInvoker struct {
	Catalog     Catalog
	Versions    VersionPolicy
	Identities  identity.Resolver
	Authority   authority.Evaluator
	Approvals   authority.ApprovalGate
	Delegations authority.DelegationPolicy
	Sod         SodChecker
	Information information.Evaluator
	Bindings    binding.Provider
	Transport   binding.Executor
	Ledger      Ledger
	Evidence    evidence.Sink
	Escalation  Escalator
	IDs         IDGenerator
	// ReadWithoutGrant lets a read-class contract proceed on its assignment when no grant exists; denials still deny.
	ReadWithoutGrant bool
}

var _ Invoker = (*AbstractInvoker)(nil)

func (i *AbstractInvoker) Invoke(ctx context.Context, inv Invocation) Result {
	if i.Catalog == nil || i.Identities == nil || i.Authority == nil || i.Bindings == nil || i.Transport == nil || i.Evidence == nil || i.IDs == nil {
		return Result{Status: StatusDenied, Reason: "invoker is not fully composed", Requirement: "CHR-AUTH-010"}
	}
	if inv.AssignmentID == nil && inv.ResponsibilityID == nil {
		return Result{Status: StatusDenied, Reason: "invocation names neither assignment nor responsibility", Requirement: "CHR-EVID-001"}
	}
	contract, err := i.Catalog.Contract(ctx, inv.Namespace, inv.CapabilityID)
	if err != nil {
		return Result{Status: StatusDenied, Reason: "capability contract: " + err.Error(), Requirement: "CHR-CAP-001", Err: err}
	}
	r := &run{invoker: i, inv: inv, contract: contract}
	if !contract.Constraints.ApprovalRequired {
		r.result.Approval = authority.ApprovalDecision{Result: authority.ApprovalResult(model.ApprovalNotRequired), Reason: "contract requires no approval"}
	}
	return r.execute(ctx)
}

type run struct {
	invoker     *AbstractInvoker
	inv         Invocation
	contract    model.CapabilityContract
	result      Result
	evidenceIDs []model.Ref
}

func (r *run) execute(ctx context.Context) Result {
	i, inv, contract := r.invoker, r.inv, r.contract
	versions := i.Versions
	if versions == nil {
		versions = BaseVersionPolicy{}
	}
	if err := versions.Compatible(inv.ContractVersion, contract.ContractVersion); err != nil {
		return r.deny(ctx, err.Error(), "CHR-CAP-008")
	}
	if _, err := i.Identities.ActiveAt(ctx, inv.Namespace, inv.Actor, inv.At); err != nil {
		return r.deny(ctx, "actor: "+err.Error(), "CHR-ID-005")
	}
	r.result.Authority = i.Authority.Evaluate(ctx, authority.Request{Namespace: inv.Namespace, EnterpriseID: inv.EnterpriseID, Actor: inv.Actor,
		CapabilityID: inv.CapabilityID, ResourceScope: inv.ResourceScope, At: inv.At, OrganizationalContext: inv.OrganizationalContext, Measures: inv.Measures,
		AuthorityChain: inv.AuthorityChain})
	switch r.result.Authority.Result {
	case authority.Allowed:
	case authority.Missing:
		if !(i.ReadWithoutGrant && contract.OperationClass == model.OperationRead && inv.AssignmentID != nil) {
			return r.deny(ctx, "authority: "+r.result.Authority.Reason, "CHR-AUTH-002")
		}
	default:
		return r.deny(ctx, "authority: "+r.result.Authority.Reason, "CHR-AUTH-010")
	}
	if contract.Constraints.ApprovalRequired || i.Delegations.ApprovalRequired(inv.AuthorityChain) {
		if i.Approvals == nil {
			return r.deny(ctx, "approval required but no approval gate is composed", "CHR-AUTH-009")
		}
		if inv.MaterialInputsDigest == "" || len(inv.SubjectRefs) == 0 {
			return r.deny(ctx, "approval requires material inputs digest and subjects", "CHR-AUTH-009")
		}
		action := inv.ApprovedAction
		if action == "" {
			action = inv.CapabilityID.ID
		}
		r.result.Approval = i.Approvals.Evaluate(ctx, authority.ApprovalRequest{Namespace: inv.Namespace, EnterpriseID: inv.EnterpriseID, Actor: inv.Actor,
			ApprovedAction: action, MaterialInputsDigest: inv.MaterialInputsDigest, SubjectRefs: inv.SubjectRefs, At: inv.At, Measures: inv.Measures,
			AuthorityChain: inv.AuthorityChain})
		if r.result.Approval.Result != authority.Approved {
			return r.deny(ctx, "approval: "+r.result.Approval.Reason, "CHR-AUTH-009")
		}
	}
	if len(contract.Constraints.SodConstraintIDs) > 0 {
		if i.Sod == nil {
			return r.deny(ctx, "separation of duties declared but no checker is composed", "CHR-AUTH-007")
		}
		ref, conflict, err := i.Sod.Conflict(ctx, inv, contract.Constraints.SodConstraintIDs)
		if err != nil {
			return r.deny(ctx, "separation of duties: "+err.Error(), "CHR-AUTH-007")
		}
		if conflict {
			r.result.SodConflict = &ref
			return r.deny(ctx, "separation of duties: conflicts with "+ref.ID, "CHR-AUTH-007")
		}
	}
	if len(inv.InformationUses) > 0 {
		if i.Information == nil {
			return r.deny(ctx, "information use declared but no governance evaluator is composed", "CHR-INFO-008")
		}
		for _, use := range inv.InformationUses {
			d := i.Information.Use(ctx, information.UseRequest{Namespace: inv.Namespace, Information: use.Information, Purpose: use.Purpose, At: inv.At})
			r.result.Information = append(r.result.Information, d)
			if d.Result != information.Allowed {
				return r.deny(ctx, fmt.Sprintf("information %s for %q: %s", use.Information.ID, use.Purpose, d.Reason), d.Requirement)
			}
		}
	}
	bind, err := (binding.Lookup{Provider: i.Bindings}).CapabilityBindingFor(ctx, inv.Namespace, inv.CapabilityID, inv.SystemProfileID)
	if err != nil {
		return r.deny(ctx, "binding: "+err.Error(), "CHR-BIND-009")
	}
	r.result.Binding = &model.Ref{Namespace: bind.Namespace, ID: bind.ID}
	if err := binding.Features(bind.FeatureSupport).Require(inv.RequiredFeatures...); err != nil {
		return r.deny(ctx, "binding "+bind.ID+": "+err.Error(), "CHR-BIND-006")
	}
	mutating := contract.Idempotency.Mutating
	if mutating {
		if inv.IdempotencyKey == "" {
			return r.deny(ctx, "mutating capability requires an idempotency key", "CHR-CAP-004")
		}
		if i.Ledger == nil {
			return r.deny(ctx, "mutating capability requires an idempotency ledger", "CHR-SEC-008")
		}
		entry, err := i.Ledger.Begin(ctx, inv.IdempotencyKey)
		if err != nil {
			return r.deny(ctx, "idempotency ledger: "+err.Error(), "CHR-SEC-008")
		}
		switch entry.State {
		case LedgerCompleted:
			if entry.Result == nil {
				return r.finish(ctx, StatusUnknown, "unknown", fmt.Sprintf("idempotency key %s is completed without a stored result; reconcile before retrying", inv.IdempotencyKey), "CHR-SEC-008")
			}
			prior := *entry.Result
			prior.Reason = fmt.Sprintf("replay of idempotency key %s; prior result returned without execution", inv.IdempotencyKey)
			return prior
		case LedgerInFlight, LedgerUnknown:
			return r.finish(ctx, StatusUnknown, "unknown", fmt.Sprintf("idempotency key %s is %s; reconcile before retrying", inv.IdempotencyKey, entry.State), "CHR-SEC-008")
		}
	}
	resp, err := i.Transport.Execute(ctx, binding.Request{Capability: inv.CapabilityID, ContractVersion: contract.ContractVersion, Inputs: inv.Inputs,
		IdempotencyKey: inv.IdempotencyKey, RequiredFeatures: inv.RequiredFeatures, SubjectRefs: inv.SubjectRefs})
	if err != nil {
		r.result.Err = err
		if errors.Is(err, binding.ErrNotExecuted) {
			r.ledger(ctx, mutating, i.Ledger.Release)
			return r.finish(ctx, StatusFailed, "failed", err.Error(), "CHR-SEC-007")
		}
		r.ledger(ctx, mutating, i.Ledger.MarkUnknown)
		return r.finish(ctx, StatusUnknown, "unknown", err.Error(), "CHR-SEC-008")
	}
	r.result.Outputs, r.result.ExternalReference = resp.Outputs, resp.ExternalReference
	r.evidenceIDs = resp.EvidenceRecordIDs
	if resp.BusinessError != "" {
		if !slices.Contains(contract.BusinessErrors, resp.BusinessError) {
			r.ledger(ctx, mutating, i.Ledger.MarkUnknown)
			return r.finish(ctx, StatusUnknown, "unknown", "undeclared business error: "+resp.BusinessError, "CHR-CAP-005")
		}
		r.result.BusinessError = resp.BusinessError
		return r.complete(ctx, mutating, StatusBusinessError, resp.BusinessError, "business error: "+resp.BusinessError)
	}
	if !slices.Contains(contract.Outcomes, resp.Outcome) {
		r.ledger(ctx, mutating, i.Ledger.MarkUnknown)
		return r.finish(ctx, StatusUnknown, "unknown", "undeclared outcome: "+resp.Outcome, "CHR-CAP-001")
	}
	r.result.Outcome = resp.Outcome
	return r.complete(ctx, mutating, StatusExecuted, resp.Outcome, "")
}

func (r *run) deny(ctx context.Context, reason, requirement string) Result {
	return r.finish(ctx, StatusDenied, "denied", reason, requirement)
}

func (r *run) complete(ctx context.Context, mutating bool, status Status, outcome, reason string) Result {
	res := r.finish(ctx, status, outcome, reason, "")
	if mutating {
		if err := r.invoker.Ledger.Complete(ctx, r.inv.IdempotencyKey, res); err != nil {
			res.Err = errors.Join(res.Err, err)
		}
	}
	return res
}

func (r *run) ledger(ctx context.Context, mutating bool, op func(context.Context, string) error) {
	if mutating {
		if err := op(ctx, r.inv.IdempotencyKey); err != nil {
			r.result.Err = errors.Join(r.result.Err, err)
		}
	}
}

// finish records the action with its authority and approval evaluations, then returns the result (CHR-EVID-001, CHR-EVID-002).
func (r *run) finish(ctx context.Context, status Status, outcome, reason, requirement string) Result {
	r.result.Status, r.result.Reason, r.result.Requirement = status, reason, requirement
	inv := r.inv
	record := model.ActionRecord{Actor: inv.Actor, RuntimeContext: inv.Runtime, AssignmentID: inv.AssignmentID, ResponsibilityID: inv.ResponsibilityID,
		CapabilityID: inv.CapabilityID, MaterialInputsDigest: inv.MaterialInputsDigest, SubjectRefs: inv.SubjectRefs, OperationClass: r.contract.OperationClass,
		ActionTime: inv.At, Outcome: clip(outcome), Disposition: string(status), AuthorityEvaluations: []model.AuthorityEvaluation{authorityEvaluation(r.result.Authority)},
		ApprovalEvaluations: []model.ApprovalEvaluation{approvalEvaluation(r.result.Approval)}, EvidenceRecordIDs: r.evidenceIDs}
	if r.result.Approval.ApprovalRef != nil {
		record.ApprovalIDs = []model.Ref{*r.result.Approval.ApprovalRef}
	}
	record.CharterSpecVersion, record.Namespace, record.ID, record.Kind = r.contract.CharterSpecVersion, inv.Namespace, r.invoker.IDs.NewID(model.KindActionRecord), model.KindActionRecord
	if inv.EnterpriseID.ID != "" {
		enterprise := inv.EnterpriseID
		record.EnterpriseID = &enterprise
	}
	for k, d := range r.result.Information {
		record.InformationEvaluations = append(record.InformationEvaluations, model.InformationEvaluation{InformationID: r.inv.InformationUses[k].Information,
			Purpose: clip(r.inv.InformationUses[k].Purpose), Result: string(d.Result), Reason: clip(d.Reason)})
	}
	if err := r.invoker.Evidence.Action(ctx, record); err != nil {
		r.result.Err = errors.Join(r.result.Err, fmt.Errorf("evidence: %w", err))
	}
	r.result.Action = &record
	r.escalate(ctx)
	return r.result
}

// escalate appends the exception and escalation for a stopped action when an Escalator is composed (CHR-AGENT-004).
func (r *run) escalate(ctx context.Context) {
	e := r.invoker.Escalation
	if e == nil || !e.Applies(r.result.Status) {
		return
	}
	exception, escalation, err := e.Escalate(ctx, r.inv, r.result)
	if err != nil {
		r.result.Err = errors.Join(r.result.Err, fmt.Errorf("escalation: %w", err))
		return
	}
	if err := r.invoker.Evidence.Exception(ctx, exception); err != nil {
		r.result.Err = errors.Join(r.result.Err, fmt.Errorf("escalation: %w", err))
		return
	}
	if err := r.invoker.Evidence.Escalation(ctx, escalation); err != nil {
		r.result.Err = errors.Join(r.result.Err, fmt.Errorf("escalation: %w", err))
		return
	}
	r.result.Exception = &model.Ref{Namespace: exception.Namespace, ID: exception.ID}
	r.result.Escalation = &model.Ref{Namespace: escalation.Namespace, ID: escalation.ID}
}

func authorityEvaluation(d authority.Decision) model.AuthorityEvaluation {
	if d.Result == "" {
		return model.AuthorityEvaluation{Result: model.AuthorityError, Reason: "not evaluated"}
	}
	return model.AuthorityEvaluation{AuthorityGrantID: d.GrantRef, Result: string(d.Result), Reason: clip(d.Reason)}
}

func approvalEvaluation(d authority.ApprovalDecision) model.ApprovalEvaluation {
	switch d.Result {
	case "":
		return model.ApprovalEvaluation{Result: model.ApprovalMissing, Reason: "not evaluated"}
	case authority.ApprovalError:
		return model.ApprovalEvaluation{Result: model.ApprovalMissing, Reason: clip("evaluation error: " + d.Reason)}
	}
	return model.ApprovalEvaluation{ApprovalID: d.ApprovalRef, ApprovedAction: d.ApprovedAction, Result: string(d.Result), Reason: clip(d.Reason)}
}

// clip keeps reasons and outcomes within the schema's 500-character limit.
func clip(s string) string {
	if len(s) > 500 {
		return s[:497] + "..."
	}
	return s
}
