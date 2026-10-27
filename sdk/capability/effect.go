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
	if rec.Action.ID == "" {
		return Result{Status: StatusDenied, Reason: "reconciliation names no action", Requirement: "CHR-EVID-012"}
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
	case LedgerInFlight:
		if held == "" {
			return r.deny(ctx, fmt.Sprintf("idempotency key %s is in flight; only its claim holder may reconcile it", key), "CHR-SEC-008")
		}
	case LedgerUnknown:
		if held == "" {
			if held, err = i.Ledger.ReclaimUnknown(ctx, key); err != nil {
				r.result.Err = err
				return r.deny(ctx, "idempotency ledger: "+err.Error(), "CHR-SEC-008")
			}
			r.result.LedgerFence = held
		}
	case LedgerCompleted:
		if entry.Result != nil {
			return replay(*entry.Result, key)
		}
		return r.deny(ctx, fmt.Sprintf("idempotency key %s is completed without a stored result", key), "CHR-SEC-008")
	default:
		r.result.LedgerFence = entry.Fence
		r.ledger(ctx, true, true, Ledger.Release)
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
	effect, violated := effectOf(judged, r.result.Postconditions)
	switch {
	case effect == model.EffectAbsent || effect == model.EffectPartial && outcome != "":
		r.setEffect(effect, judged)
		return r.effectAbsent(ctx, violated)
	case outcome == "" || len(judged) == 0 || effect != model.EffectComplete:
		// Without a named outcome, a mix of held and missing effects does not say which outcome was reached.
		if effect == model.EffectUnobserved {
			r.setEffect(effect, judged)
		}
		return r.finish(ctx, StatusUnknown, "unknown", "observation does not establish a declared outcome; the attempt remains unknown", "CHR-EVID-012")
	}
	r.setEffect(effect, judged)
	r.result.Outcome = outcome
	return r.complete(ctx, true, StatusExecuted, outcome, fmt.Sprintf("reconciled %s from observed effects without executing again", r.reconciles.ID), "")
}

// verifyEffect observes the postconditions of an accepted mutation; false stops the run short of executed (CHR-CAP-010).
func (r *invocationRun) verifyEffect(ctx context.Context, outcome string) (Result, bool) {
	if r.invoker.Observer == nil || !slices.ContainsFunc(r.contract.Postconditions, func(p model.Postcondition) bool { return p.AppliesTo(outcome) }) {
		return Result{}, true
	}
	r.observe(ctx, outcome)
	required := r.contract.RequiredPostconditions(outcome)
	effect, violated := effectOf(required, r.result.Postconditions)
	r.setEffect(effect, required)
	switch effect {
	case model.EffectComplete:
		return Result{}, true
	case model.EffectAbsent, model.EffectPartial:
		return r.effectAbsent(ctx, violated), false
	}
	r.ledger(ctx, true, false, Ledger.MarkUnknown)
	return r.finish(ctx, StatusUnknown, "unknown", "the mutation was accepted but its effect could not be observed; reconcile before retrying", "CHR-CAP-010"), false
}

// effectAbsent records a violated postcondition as its declared business error, otherwise as failed. The key is
// released for a retry only when the contract declares one safe; otherwise it replays the failure. A partial effect
// is failed, never retried, and escalated (CHR-EVID-016).
func (r *invocationRun) effectAbsent(ctx context.Context, p model.Postcondition) Result {
	if r.result.Effect == model.EffectPartial {
		return r.complete(ctx, true, StatusFailed, "failed", fmt.Sprintf("postcondition %s is violated while others hold: the effect is partial", p.ID), "CHR-EVID-016")
	}
	reason := fmt.Sprintf("postcondition %s is violated: the effect was not observed", p.ID)
	if slices.Contains(r.contract.BusinessErrors, p.ViolationBusinessError) {
		r.result.BusinessError = p.ViolationBusinessError
		return r.complete(ctx, true, StatusBusinessError, p.ViolationBusinessError, reason, "CHR-CAP-010")
	}
	if r.contract.Idempotency.RetryWhenEffectAbsent {
		if r.reconciles == nil {
			r.ledger(ctx, true, true, Ledger.Release)
		} else if err := r.invoker.Ledger.Release(ctx, r.inv.IdempotencyKey, r.result.LedgerFence); err != nil {
			return r.uncommitted(ctx, err)
		} else {
			r.result.LedgerFence = ""
		}
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
	r.causes = map[string]string{}
	for _, p := range posts {
		e, cause := r.evaluate(ctx, p, obs, err)
		if e.Reason == "" {
			e.Reason = e.Result
		}
		if e.ObservedAt.IsZero() {
			e.ObservedAt = inv.At
		}
		e.Reason = clip(e.Reason)
		r.cite(e.EvidenceRecordIDs)
		r.causes[p.ID] = cause
		r.result.Postconditions = append(r.result.Postconditions, e)
	}
	if err != nil {
		return ""
	}
	return obs.Outcome
}

// evaluate settles one postcondition and, when unknown, why. A predicate decides it from the observed facts; an
// observer result that disagrees with the predicate or with itself makes it contradictory.
func (r *invocationRun) evaluate(ctx context.Context, p model.Postcondition, obs binding.Observation, err error) (model.PostconditionEvaluation, string) {
	unknown := func(reason, cause string) (model.PostconditionEvaluation, string) {
		return model.PostconditionEvaluation{PostconditionID: p.ID, Result: model.PostconditionUnknown, Reason: reason}, cause
	}
	if err != nil {
		return unknown("observation failed: "+err.Error(), model.UnknownObservationFailed)
	}
	var e *model.PostconditionEvaluation
	for k, o := range obs.Evaluations {
		if o.PostconditionID != p.ID {
			continue
		}
		if e != nil && e.Result != o.Result {
			return unknown("the observer reported conflicting results", model.UnknownContradictory)
		}
		e = &obs.Evaluations[k]
	}
	if p.Predicate != nil {
		result, reason := r.decide(ctx, *p.Predicate, obs)
		if e != nil && decisive(e.Result) && decisive(result) && e.Result != result {
			return unknown("the observer reported "+e.Result+" but the predicate is "+result, model.UnknownContradictory)
		}
		e = &model.PostconditionEvaluation{PostconditionID: p.ID, Result: result, Reason: reason, EvidenceRecordIDs: obs.EvidenceRecordIDs}
	}
	switch {
	case e == nil:
		return unknown("not evaluated by the observer", model.UnknownUnevidenced)
	case !decisive(e.Result):
		out := *e
		out.Result = model.PostconditionUnknown
		return out, model.UnknownUnevidenced
	case len(e.EvidenceRecordIDs) == 0:
		return unknown("observer reported "+e.Result+" without observed-fact evidence", model.UnknownUnevidenced)
	}
	return *e, ""
}

func decisive(result string) bool {
	return result == model.PostconditionSatisfied || result == model.PostconditionViolated
}

// cite adds evidence the action relies on, once each.
func (r *invocationRun) cite(ids []model.Ref) {
	for _, id := range ids {
		if !slices.Contains(r.evidenceIDs, id) {
			r.evidenceIDs = append(r.evidenceIDs, id)
		}
	}
}

// setEffect records the effect of the judged postconditions and, when unobserved, the most telling cause: a failed
// observation, then a contradiction, then missing evidence.
func (r *invocationRun) setEffect(effect string, judged []model.Postcondition) {
	if len(judged) == 0 {
		return
	}
	r.result.Effect, r.result.UnknownCause = effect, ""
	if effect != model.EffectUnobserved {
		return
	}
	for _, cause := range []string{model.UnknownObservationFailed, model.UnknownContradictory, model.UnknownUnevidenced} {
		for _, p := range judged {
			if r.causes[p.ID] == cause {
				r.result.UnknownCause = cause
				return
			}
		}
	}
	r.result.UnknownCause = model.UnknownUnevidenced
}

// effectOf classifies the evaluations of the required postconditions posts (CHR-EVID-016) and returns the first
// violated one.
func effectOf(posts []model.Postcondition, evals []model.PostconditionEvaluation) (string, model.Postcondition) {
	var violated *model.Postcondition
	satisfied := 0
	for _, p := range posts {
		k := slices.IndexFunc(evals, func(e model.PostconditionEvaluation) bool { return e.PostconditionID == p.ID })
		switch {
		case k < 0 || evals[k].Result == model.PostconditionUnknown:
			return model.EffectUnobserved, model.Postcondition{}
		case evals[k].Result == model.PostconditionViolated:
			if violated == nil {
				violated = &p
			}
		default:
			satisfied++
		}
	}
	switch {
	case violated == nil:
		return model.EffectComplete, model.Postcondition{}
	case satisfied == 0:
		return model.EffectAbsent, *violated
	}
	return model.EffectPartial, *violated
}
