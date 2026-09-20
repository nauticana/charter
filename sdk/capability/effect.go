package capability

import (
	"context"
	"fmt"
	"slices"

	"github.com/nauticana/charter/sdk/binding"
	"github.com/nauticana/charter/sdk/model"
)

var _ Reconciler = (*BaseInvoker)(nil)

// Reconcile re-runs every gate, then settles the key from observed effects: executed when the observer names a
// declared outcome whose required postconditions all hold, the absent-effect disposition when they are violated, and
// unknown otherwise. The earlier record is left as written; the new one names it (CHR-EVID-004).
func (i *BaseInvoker) Reconcile(ctx context.Context, rec Reconciliation) Result {
	r, refused := i.newInvocationRun(ctx, rec.Invocation)
	if r == nil {
		return refused
	}
	if rec.Action.ID == "" || rec.Fence == "" {
		return Result{Status: StatusDenied, Reason: "reconciliation names no action or holds no ledger fence", Requirement: "CHR-EVID-012"}
	}
	action := rec.Action
	r.reconciles, r.result.LedgerFence = &action, rec.Fence
	return r.reconcile(ctx)
}

func (r *invocationRun) reconcile(ctx context.Context) Result {
	if denied, ok := r.gate(ctx); !ok {
		return denied
	}
	i, key, contract := r.invoker, r.inv.IdempotencyKey, r.contract
	required := contract.RequiredPostconditions("")
	if !contract.Idempotency.Mutating || len(required) == 0 || key == "" || i.Ledger == nil {
		return r.deny(ctx, "reconciliation needs a mutating capability with required postconditions, an idempotency key, and a ledger", "CHR-EVID-012")
	}
	held := r.result.LedgerFence
	entry, err := i.Ledger.Begin(ctx, key)
	if err != nil {
		return r.deny(ctx, "idempotency ledger: "+err.Error(), "CHR-SEC-008")
	}
	switch entry.State {
	case LedgerInFlight, LedgerUnknown:
	case LedgerCompleted:
		if entry.Result != nil {
			return replay(*entry.Result, key)
		}
		return r.deny(ctx, fmt.Sprintf("idempotency key %s is completed without a stored result", key), "CHR-SEC-008")
	default:
		r.result.LedgerFence = entry.Fence
		r.ledger(ctx, true, true, i.Ledger.Release)
		r.result.LedgerFence = held
		return r.deny(ctx, fmt.Sprintf("idempotency key %s has no attempt to reconcile", key), "CHR-EVID-012")
	}
	// Marking the key unknown proves the fence before anything is observed or recorded.
	if err := i.Ledger.MarkUnknown(ctx, key, held); err != nil {
		r.result.Err = err
		return r.deny(ctx, "idempotency ledger: "+err.Error(), "CHR-SEC-008")
	}
	outcome := r.observe(ctx, "")
	if !slices.Contains(contract.Outcomes, outcome) {
		outcome = ""
	}
	judged := contract.RequiredPostconditions(outcome)
	violated, state := verdict(judged, r.result.Postconditions)
	// With no outcome named, the effect is absent only when no required postcondition holds.
	absent := state == model.PostconditionViolated && (outcome != "" || !slices.ContainsFunc(r.result.Postconditions, func(e model.PostconditionEvaluation) bool {
		return e.Result == model.PostconditionSatisfied && slices.ContainsFunc(judged, func(p model.Postcondition) bool { return p.ID == e.PostconditionID })
	}))
	switch {
	case absent:
		return r.effectAbsent(ctx, violated)
	case outcome == "" || len(judged) == 0 || state != model.PostconditionSatisfied:
		return r.finish(ctx, StatusUnknown, "unknown", "observation does not establish a declared outcome; the attempt remains unknown", "CHR-EVID-012")
	}
	r.result.Outcome = outcome
	return r.complete(ctx, true, StatusExecuted, outcome, fmt.Sprintf("reconciled %s from observed effects without executing again", r.reconciles.ID), "")
}

// verifyEffect observes the postconditions of an accepted mutation; false stops the run short of executed (CHR-CAP-010).
func (r *invocationRun) verifyEffect(ctx context.Context, outcome string) (Result, bool) {
	if r.invoker.Observer == nil || !slices.ContainsFunc(r.contract.Postconditions, func(p model.Postcondition) bool { return p.AppliesTo(outcome) }) {
		return Result{}, true
	}
	r.observe(ctx, outcome)
	switch violated, state := verdict(r.contract.RequiredPostconditions(outcome), r.result.Postconditions); state {
	case model.PostconditionSatisfied:
		return Result{}, true
	case model.PostconditionViolated:
		return r.effectAbsent(ctx, violated), false
	default:
		r.ledger(ctx, true, false, r.invoker.Ledger.MarkUnknown)
		return r.finish(ctx, StatusUnknown, "unknown", "the mutation was accepted but its effect could not be observed; reconcile before retrying", "CHR-CAP-010"), false
	}
}

// effectAbsent records a violated postcondition as its declared business error, otherwise as failed. The key is
// released for a retry only when the contract declares one safe; otherwise it replays the failure.
func (r *invocationRun) effectAbsent(ctx context.Context, p model.Postcondition) Result {
	reason := fmt.Sprintf("postcondition %s is violated: the effect was not observed", p.ID)
	if slices.Contains(r.contract.BusinessErrors, p.ViolationBusinessError) {
		r.result.BusinessError = p.ViolationBusinessError
		return r.complete(ctx, true, StatusBusinessError, p.ViolationBusinessError, reason, "CHR-CAP-010")
	}
	if r.contract.Idempotency.RetryWhenEffectAbsent {
		r.ledger(ctx, true, true, r.invoker.Ledger.Release)
		return r.finish(ctx, StatusFailed, "failed", reason, "CHR-CAP-010")
	}
	return r.complete(ctx, true, StatusFailed, "failed", reason, "CHR-CAP-010")
}

// observe records one evaluation per postcondition applying to outcome and returns the outcome the observer named.
// Anything short of an evidenced satisfied or violated result is unknown (CHR-EVID-011).
func (r *invocationRun) observe(ctx context.Context, outcome string) string {
	inv := r.inv
	var posts []model.Postcondition
	for _, p := range r.contract.Postconditions {
		if p.AppliesTo(outcome) {
			posts = append(posts, p)
		}
	}
	obs, err := r.invoker.Observer.Observe(ctx, binding.ObservationRequest{Capability: inv.CapabilityID, ContractVersion: r.contract.ContractVersion,
		Inputs: inv.Inputs, IdempotencyKey: inv.IdempotencyKey, SubjectRefs: inv.SubjectRefs, ExternalReference: r.result.ExternalReference,
		Outcome: outcome, Postconditions: posts})
	for _, p := range posts {
		e := model.PostconditionEvaluation{PostconditionID: p.ID, Result: model.PostconditionUnknown, Reason: "not evaluated by the observer"}
		if k := slices.IndexFunc(obs.Evaluations, func(o model.PostconditionEvaluation) bool { return o.PostconditionID == p.ID }); err == nil && k >= 0 {
			e = obs.Evaluations[k]
		}
		switch {
		case err != nil:
			e.Reason = "observation failed: " + err.Error()
		case e.Result != model.PostconditionSatisfied && e.Result != model.PostconditionViolated:
			e.Result = model.PostconditionUnknown
		case len(e.EvidenceRecordIDs) == 0:
			e.Result, e.Reason = model.PostconditionUnknown, "observer reported "+e.Result+" without observed-fact evidence"
		}
		if e.Reason == "" {
			e.Reason = e.Result
		}
		if e.ObservedAt.IsZero() {
			e.ObservedAt = inv.At
		}
		e.Reason = clip(e.Reason)
		for _, id := range e.EvidenceRecordIDs {
			if !slices.Contains(r.evidenceIDs, id) {
				r.evidenceIDs = append(r.evidenceIDs, id)
			}
		}
		r.result.Postconditions = append(r.result.Postconditions, e)
	}
	if err != nil {
		return ""
	}
	return obs.Outcome
}

// verdict folds the evaluations of posts: unknown outranks violated, which outranks satisfied; the violated
// postcondition is returned with that state.
func verdict(posts []model.Postcondition, evals []model.PostconditionEvaluation) (model.Postcondition, string) {
	var violated *model.Postcondition
	for _, p := range posts {
		k := slices.IndexFunc(evals, func(e model.PostconditionEvaluation) bool { return e.PostconditionID == p.ID })
		switch {
		case k < 0 || evals[k].Result == model.PostconditionUnknown:
			return model.Postcondition{}, model.PostconditionUnknown
		case evals[k].Result == model.PostconditionViolated && violated == nil:
			violated = &p
		}
	}
	if violated != nil {
		return *violated, model.PostconditionViolated
	}
	return model.Postcondition{}, model.PostconditionSatisfied
}
