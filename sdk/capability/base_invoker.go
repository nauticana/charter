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

// BaseInvoker runs the governed invocation pipeline around an abstract binding.Executor: contract and version,
// actor lifecycle, authority, approval, separation of duties, information governance, binding feature support,
// idempotency, execution, outcome and effect verification, evidence, and escalation of stopped actions. Every gate fails closed
// and every governed decision is recorded (CHR-AUTH-010, CHR-AGENT-008).
type BaseInvoker struct {
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
	// Observer reads external effects; a mutating contract with required postconditions is denied without one.
	Observer   binding.Observer
	Ledger     Ledger
	Evidence   evidence.Sink
	Escalation Escalator
	IDs        IDGenerator
	// ReadWithoutGrant lets a read-class contract proceed on its assignment when no grant exists; denials still deny.
	ReadWithoutGrant bool
}

var _ Invoker = (*BaseInvoker)(nil)

func (i *BaseInvoker) Invoke(ctx context.Context, inv Invocation) Result {
	r, refused := i.newInvocationRun(ctx, inv)
	if r == nil {
		return refused
	}
	return r.execute(ctx)
}

// newInvocationRun refuses, without evidence, an invocation that cannot be attributed or whose contract cannot be read.
func (i *BaseInvoker) newInvocationRun(ctx context.Context, inv Invocation) (*invocationRun, Result) {
	if i.Catalog == nil || i.Identities == nil || i.Authority == nil || i.Bindings == nil || i.Transport == nil || i.Evidence == nil || i.IDs == nil {
		return nil, Result{Status: StatusDenied, Reason: "invoker is not fully composed", Requirement: "CHR-AUTH-010"}
	}
	if inv.AssignmentID == nil && inv.ResponsibilityID == nil {
		return nil, Result{Status: StatusDenied, Reason: "invocation names neither assignment nor responsibility", Requirement: "CHR-EVID-001"}
	}
	contract, err := i.Catalog.Contract(ctx, inv.Namespace, inv.CapabilityID)
	if err != nil {
		return nil, Result{Status: StatusDenied, Reason: "capability contract: " + err.Error(), Requirement: "CHR-CAP-001", Err: err}
	}
	r := &invocationRun{invoker: i, inv: inv, contract: contract}
	if !contract.Constraints.ApprovalRequired {
		r.result.Approval = authority.ApprovalDecision{Result: authority.ApprovalResult(model.ApprovalNotRequired), Reason: "contract requires no approval"}
	}
	return r, Result{}
}

// invocationRun is the state of one pass of an invocation through the pipeline: the decisions made so far and the evidence they cite.
type invocationRun struct {
	invoker     *BaseInvoker
	inv         Invocation
	contract    model.CapabilityContract
	result      Result
	evidenceIDs []model.Ref
	reconciles  *model.Ref
}

// gate runs every decision that precedes the external system; false stops the run with the recorded denial.
func (r *invocationRun) gate(ctx context.Context) (Result, bool) {
	i, inv, contract := r.invoker, r.inv, r.contract
	versions := i.Versions
	if versions == nil {
		versions = BaseVersionPolicy{}
	}
	if err := versions.Compatible(inv.ContractVersion, contract.ContractVersion); err != nil {
		return r.deny(ctx, err.Error(), "CHR-CAP-008"), false
	}
	if _, err := i.Identities.ActiveAt(ctx, inv.Namespace, inv.Actor, inv.At); err != nil {
		return r.deny(ctx, "actor: "+err.Error(), "CHR-ID-005"), false
	}
	r.result.Authority = i.Authority.Evaluate(ctx, authority.Request{Namespace: inv.Namespace, EnterpriseID: inv.EnterpriseID, Actor: inv.Actor,
		CapabilityID: inv.CapabilityID, ResourceScope: inv.ResourceScope, At: inv.At, OrganizationalContext: inv.OrganizationalContext, Measures: inv.Measures,
		AuthorityChain: inv.AuthorityChain})
	switch r.result.Authority.Result {
	case authority.Allowed:
	case authority.Missing:
		if !(i.ReadWithoutGrant && contract.OperationClass == model.OperationRead && inv.AssignmentID != nil) {
			return r.deny(ctx, "authority: "+r.result.Authority.Reason, "CHR-AUTH-002"), false
		}
	default:
		return r.deny(ctx, "authority: "+r.result.Authority.Reason, "CHR-AUTH-010"), false
	}
	if contract.Constraints.ApprovalRequired || i.Delegations.ApprovalRequired(inv.AuthorityChain) {
		if i.Approvals == nil {
			return r.deny(ctx, "approval required but no approval gate is composed", "CHR-AUTH-009"), false
		}
		if inv.MaterialInputsDigest == "" || len(inv.SubjectRefs) == 0 {
			return r.deny(ctx, "approval requires material inputs digest and subjects", "CHR-AUTH-009"), false
		}
		action := inv.ApprovedAction
		if action == "" {
			action = inv.CapabilityID.ID
		}
		r.result.Approval = i.Approvals.Evaluate(ctx, authority.ApprovalRequest{Namespace: inv.Namespace, EnterpriseID: inv.EnterpriseID, Actor: inv.Actor,
			ApprovedAction: action, MaterialInputsDigest: inv.MaterialInputsDigest, SubjectRefs: inv.SubjectRefs, At: inv.At, Measures: inv.Measures,
			AuthorityChain: inv.AuthorityChain})
		if r.result.Approval.Result != authority.Approved {
			return r.deny(ctx, "approval: "+r.result.Approval.Reason, "CHR-AUTH-009"), false
		}
	}
	if len(contract.Constraints.SodConstraintIDs) > 0 {
		if i.Sod == nil {
			return r.deny(ctx, "separation of duties declared but no checker is composed", "CHR-AUTH-007"), false
		}
		ref, conflict, err := i.Sod.Conflict(ctx, inv, contract.Constraints.SodConstraintIDs)
		if err != nil {
			return r.deny(ctx, "separation of duties: "+err.Error(), "CHR-AUTH-007"), false
		}
		if conflict {
			r.result.SodConflict = &ref
			return r.deny(ctx, "separation of duties: conflicts with "+ref.ID, "CHR-AUTH-007"), false
		}
	}
	if len(inv.InformationUses) > 0 {
		if i.Information == nil {
			return r.deny(ctx, "information use declared but no governance evaluator is composed", "CHR-INFO-008"), false
		}
		for _, use := range inv.InformationUses {
			d := i.Information.Use(ctx, information.UseRequest{Namespace: inv.Namespace, Information: use.Information, Purpose: use.Purpose, At: inv.At})
			r.result.Information = append(r.result.Information, d)
			if d.Result != information.Allowed {
				return r.deny(ctx, fmt.Sprintf("information %s for %q: %s", use.Information.ID, use.Purpose, d.Reason), d.Requirement), false
			}
		}
	}
	bind, err := (binding.Lookup{Provider: i.Bindings}).CapabilityBindingFor(ctx, inv.Namespace, inv.CapabilityID, inv.SystemProfileID)
	if err != nil {
		return r.deny(ctx, "binding: "+err.Error(), "CHR-BIND-009"), false
	}
	r.result.Binding = &model.Ref{Namespace: bind.Namespace, ID: bind.ID}
	if err := binding.Features(bind.FeatureSupport).Require(inv.RequiredFeatures...); err != nil {
		return r.deny(ctx, "binding "+bind.ID+": "+err.Error(), "CHR-BIND-006"), false
	}
	mutating := contract.Idempotency.Mutating
	if mutating && len(contract.RequiredPostconditions("")) > 0 && i.Observer == nil {
		return r.deny(ctx, "capability declares required postconditions but no effect observer is composed", "CHR-CAP-010"), false
	}
	return Result{}, true
}

func (r *invocationRun) execute(ctx context.Context) Result {
	if denied, ok := r.gate(ctx); !ok {
		return denied
	}
	i, inv, contract := r.invoker, r.inv, r.contract
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
		case LedgerNew:
			if entry.Fence == "" {
				return r.finish(ctx, StatusUnknown, "unknown", fmt.Sprintf("idempotency key %s was claimed without a fence; reconcile before retrying", inv.IdempotencyKey), "CHR-SEC-008")
			}
			r.result.LedgerFence = entry.Fence
		case LedgerCompleted:
			if entry.Result == nil {
				return r.finish(ctx, StatusUnknown, "unknown", fmt.Sprintf("idempotency key %s is completed without a stored result; reconcile before retrying", inv.IdempotencyKey), "CHR-SEC-008")
			}
			return replay(*entry.Result, inv.IdempotencyKey)
		case LedgerInFlight, LedgerUnknown:
			return r.finish(ctx, StatusUnknown, "unknown", fmt.Sprintf("idempotency key %s is %s; reconcile before retrying", inv.IdempotencyKey, entry.State), "CHR-SEC-008")
		default:
			return r.finish(ctx, StatusUnknown, "unknown", fmt.Sprintf("idempotency key %s has invalid state %q; reconcile before retrying", inv.IdempotencyKey, entry.State), "CHR-SEC-008")
		}
	}
	resp, err := i.Transport.Execute(ctx, binding.Request{Capability: inv.CapabilityID, ContractVersion: contract.ContractVersion, Inputs: inv.Inputs,
		IdempotencyKey: inv.IdempotencyKey, RequiredFeatures: inv.RequiredFeatures, SubjectRefs: inv.SubjectRefs})
	if err != nil {
		r.result.Err = err
		if errors.Is(err, binding.ErrNotExecuted) {
			r.ledger(ctx, mutating, true, i.Ledger.Release)
			return r.finish(ctx, StatusFailed, "failed", err.Error(), "CHR-SEC-007")
		}
		r.ledger(ctx, mutating, false, i.Ledger.MarkUnknown)
		return r.finish(ctx, StatusUnknown, "unknown", err.Error(), "CHR-SEC-008")
	}
	r.result.Outputs, r.result.ExternalReference = resp.Outputs, resp.ExternalReference
	r.evidenceIDs = resp.EvidenceRecordIDs
	if resp.BusinessError != "" {
		if !slices.Contains(contract.BusinessErrors, resp.BusinessError) {
			r.ledger(ctx, mutating, false, i.Ledger.MarkUnknown)
			return r.finish(ctx, StatusUnknown, "unknown", "undeclared business error: "+resp.BusinessError, "CHR-CAP-005")
		}
		r.result.BusinessError = resp.BusinessError
		return r.complete(ctx, mutating, StatusBusinessError, resp.BusinessError, "business error: "+resp.BusinessError, "")
	}
	if !slices.Contains(contract.Outcomes, resp.Outcome) {
		r.ledger(ctx, mutating, false, i.Ledger.MarkUnknown)
		return r.finish(ctx, StatusUnknown, "unknown", "undeclared outcome: "+resp.Outcome, "CHR-CAP-001")
	}
	if mutating {
		if stopped, ok := r.verifyEffect(ctx, resp.Outcome); !ok {
			return stopped
		}
	}
	r.result.Outcome = resp.Outcome
	return r.complete(ctx, mutating, StatusExecuted, resp.Outcome, "", "")
}

func replay(prior Result, key string) Result {
	prior.LedgerFence = ""
	prior.Reason = fmt.Sprintf("replay of idempotency key %s; prior result returned without execution", key)
	return prior
}

func (r *invocationRun) deny(ctx context.Context, reason, requirement string) Result {
	return r.finish(ctx, StatusDenied, "denied", reason, requirement)
}

func (r *invocationRun) complete(ctx context.Context, mutating bool, status Status, outcome, reason, requirement string) Result {
	if mutating && r.reconciles != nil {
		// A later reclaim may supersede this run, so commit its fence before publishing a terminal record.
		r.prepare(status, outcome, reason, requirement)
		fence := r.result.LedgerFence
		stored := r.result
		stored.LedgerFence = ""
		stored.Err = nil
		if err := r.invoker.Ledger.Complete(ctx, r.inv.IdempotencyKey, fence, stored); err != nil {
			return r.uncommitted(ctx, err)
		}
		r.result.LedgerFence = ""
		return r.record(ctx)
	}
	res := r.finish(ctx, status, outcome, reason, requirement)
	if mutating {
		fence := res.LedgerFence
		stored := res
		stored.LedgerFence = ""
		stored.Err = nil
		if err := r.invoker.Ledger.Complete(ctx, r.inv.IdempotencyKey, fence, stored); err != nil {
			res.Err = errors.Join(res.Err, err)
		} else {
			res.LedgerFence = ""
		}
	}
	return res
}

// uncommitted leaves a reconciliation whose ledger write failed unknown, publishing none of its undecided outcome.
func (r *invocationRun) uncommitted(ctx context.Context, err error) Result {
	r.result.Action, r.result.Outcome, r.result.BusinessError = nil, "", ""
	r.result.Err = errors.Join(r.result.Err, err)
	return r.finish(ctx, StatusUnknown, "unknown", "the reconciliation could not commit its ledger result before publication", "CHR-SEC-008")
}

func (r *invocationRun) ledger(ctx context.Context, mutating, clearFence bool, op func(context.Context, string, string) error) {
	if mutating {
		if err := op(ctx, r.inv.IdempotencyKey, r.result.LedgerFence); err != nil {
			r.result.Err = errors.Join(r.result.Err, err)
		} else if clearFence {
			r.result.LedgerFence = ""
		}
	}
}

// finish records the action with its authority and approval evaluations, then returns the result (CHR-EVID-001, CHR-EVID-002).
func (r *invocationRun) finish(ctx context.Context, status Status, outcome, reason, requirement string) Result {
	r.prepare(status, outcome, reason, requirement)
	return r.record(ctx)
}

func (r *invocationRun) prepare(status Status, outcome, reason, requirement string) {
	r.result.Status, r.result.Reason, r.result.Requirement = status, reason, requirement
	inv := r.inv
	record := model.ActionRecord{Actor: inv.Actor, RuntimeContext: inv.Runtime, AssignmentID: inv.AssignmentID, ResponsibilityID: inv.ResponsibilityID,
		CapabilityID: inv.CapabilityID, MaterialInputsDigest: inv.MaterialInputsDigest, SubjectRefs: inv.SubjectRefs, OperationClass: r.contract.OperationClass,
		ActionTime: inv.At, Outcome: clip(outcome), Disposition: string(status), AuthorityEvaluations: []model.AuthorityEvaluation{authorityEvaluation(r.result.Authority)},
		ApprovalEvaluations: []model.ApprovalEvaluation{approvalEvaluation(r.result.Approval)}, EvidenceRecordIDs: r.evidenceIDs,
		PostconditionEvaluations: r.result.Postconditions, ReconcilesActionID: r.reconciles}
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
	r.result.Action = &record
}

func (r *invocationRun) record(ctx context.Context) Result {
	if err := r.invoker.Evidence.Action(ctx, *r.result.Action); err != nil {
		r.result.Err = errors.Join(r.result.Err, fmt.Errorf("evidence: %w", err))
	}
	r.escalate(ctx)
	return r.result
}

// escalate appends the exception and escalation for a stopped action when an Escalator is composed (CHR-AGENT-004).
func (r *invocationRun) escalate(ctx context.Context) {
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
