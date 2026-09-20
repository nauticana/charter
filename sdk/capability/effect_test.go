package capability_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/binding"
	"github.com/nauticana/charter/sdk/capability"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// amendedCatalog serves the harness contracts after amend has edited them.
type amendedCatalog struct {
	capability.Catalog
	amend func(*model.CapabilityContract)
}

func (c amendedCatalog) Contract(ctx context.Context, namespace string, ref model.Ref) (model.CapabilityContract, error) {
	contract, err := c.Catalog.Contract(ctx, namespace, ref)
	c.amend(&contract)
	return contract, err
}

func (h *harness) amend(f func(*model.CapabilityContract)) {
	h.invoker.Catalog = amendedCatalog{Catalog: h.invoker.Catalog, amend: f}
}

func TestExecutedRequiresObservedEffect(t *testing.T) {
	ctx := context.Background()
	t.Run("observed effect is recorded with its evidence", func(t *testing.T) {
		h := newHarness(t)
		rec := h.recorded(h.invoker.Invoke(ctx, reserve()))
		if rec.Disposition != model.DispositionExecuted || len(rec.PostconditionEvaluations) != 1 || rec.PostconditionEvaluations[0].Result != model.PostconditionSatisfied ||
			len(rec.EvidenceRecordIDs) != 1 || rec.EvidenceRecordIDs[0].ID != "EVR-0042-RESERVATION-OBSERVED" || h.observer.calls != 1 {
			t.Fatalf("record %+v", rec)
		}
	})
	t.Run("accepted but unchanged is failed and replays the failure", func(t *testing.T) {
		h := newHarness(t)
		h.observer.result = model.PostconditionViolated
		res := h.invoker.Invoke(ctx, reserve())
		if rec := h.recorded(res); res.Status != capability.StatusFailed || res.Requirement != "CHR-CAP-010" || res.Outcome != "" || rec.Disposition != model.DispositionFailed {
			t.Fatalf("got %s %s (%s)", res.Status, res.Requirement, res.Reason)
		}
		if again := h.invoker.Invoke(ctx, reserve()); again.Status != capability.StatusFailed || h.transport.calls != 1 {
			t.Fatalf("retry got %s after %d transport calls", again.Status, h.transport.calls)
		}
	})
	t.Run("absent effect releases the key only when the contract declares the retry safe", func(t *testing.T) {
		h := newHarness(t)
		h.amend(func(c *model.CapabilityContract) { c.Idempotency.RetryWhenEffectAbsent = true })
		h.observer.result = model.PostconditionViolated
		if res := h.invoker.Invoke(ctx, reserve()); res.Status != capability.StatusFailed || res.LedgerFence != "" {
			t.Fatalf("got %s fence %q", res.Status, res.LedgerFence)
		}
		h.observer.result = model.PostconditionSatisfied
		if res := h.invoker.Invoke(ctx, reserve()); res.Status != capability.StatusExecuted || h.transport.calls != 2 {
			t.Fatalf("retry got %s after %d transport calls", res.Status, h.transport.calls)
		}
	})
	t.Run("violation maps to its declared business error", func(t *testing.T) {
		h := newHarness(t)
		h.amend(func(c *model.CapabilityContract) { c.Postconditions[0].ViolationBusinessError = "invalid order state" })
		h.observer.result = model.PostconditionViolated
		if res := h.invoker.Invoke(ctx, reserve()); res.Status != capability.StatusBusinessError || res.BusinessError != "invalid order state" {
			t.Fatalf("got %s %q", res.Status, res.BusinessError)
		}
	})
	for name, arrange := range map[string]func(*fakeObserver){
		"observation failure":   func(o *fakeObserver) { o.err = errors.New("read timed out") },
		"unevidenced success":   func(o *fakeObserver) { o.unevidenced = true },
		"unrecognized result":   func(o *fakeObserver) { o.result = "probably" },
		"observer says unknown": func(o *fakeObserver) { o.result = model.PostconditionUnknown },
	} {
		t.Run(name+" is unknown and blocks the retry", func(t *testing.T) {
			h := newHarness(t)
			arrange(h.observer)
			res := h.invoker.Invoke(ctx, reserve())
			if rec := h.recorded(res); res.Status != capability.StatusUnknown || res.Requirement != "CHR-CAP-010" || res.LedgerFence == "" || rec.PostconditionEvaluations[0].Result != model.PostconditionUnknown {
				t.Fatalf("got %s %s fence %q", res.Status, res.Requirement, res.LedgerFence)
			}
			if again := h.invoker.Invoke(ctx, reserve()); again.Status != capability.StatusUnknown || h.transport.calls != 1 {
				t.Fatalf("retry got %s after %d transport calls", again.Status, h.transport.calls)
			}
		})
	}
	t.Run("required postconditions without an observer deny before the mutation", func(t *testing.T) {
		h := newHarness(t)
		h.invoker.Observer = nil
		if res := h.invoker.Invoke(ctx, reserve()); res.Status != capability.StatusDenied || res.Requirement != "CHR-CAP-010" || h.transport.calls != 0 {
			t.Fatalf("got %s %s after %d transport calls", res.Status, res.Requirement, h.transport.calls)
		}
	})
	t.Run("an outcome no postcondition applies to is not observed", func(t *testing.T) {
		h := newHarness(t)
		h.transport.resp = binding.Response{Outcome: "rejected"}
		if res := h.invoker.Invoke(ctx, reserve()); res.Status != capability.StatusExecuted || h.observer.calls != 0 || len(res.Postconditions) != 0 {
			t.Fatalf("got %s after %d observations", res.Status, h.observer.calls)
		}
	})
}

func TestReconciliationSettlesUnknownFromObservation(t *testing.T) {
	ctx := context.Background()
	timedOut := func(t *testing.T) (*harness, capability.Reconciliation) {
		h := newHarness(t)
		h.transport.err = binding.ErrOutcomeUnknown
		res := h.invoker.Invoke(ctx, reserve())
		if res.Status != capability.StatusUnknown || res.LedgerFence == "" {
			t.Fatalf("attempt got %s", res.Status)
		}
		h.transport.err = nil
		inv := reserve()
		inv.At = at.Add(3 * time.Minute)
		return h, capability.Reconciliation{Invocation: inv, Action: model.Ref{ID: res.Action.ID}, Fence: res.LedgerFence}
	}
	t.Run("observed effect completes the key without executing again", func(t *testing.T) {
		h, rec := timedOut(t)
		h.observer.outcome = "already-reserved"
		res := h.invoker.Reconcile(ctx, rec)
		record := h.recorded(res)
		if res.Status != capability.StatusExecuted || res.Outcome != "already-reserved" || res.LedgerFence != "" || h.transport.calls != 1 ||
			record.ReconcilesActionID == nil || record.ReconcilesActionID.ID != rec.Action.ID || record.ID == rec.Action.ID {
			t.Fatalf("got %s %q (%s) after %d transport calls", res.Status, res.Outcome, res.Reason, h.transport.calls)
		}
		prior, err := h.sink.Store.Fetch(ctx, corpus.DocumentKey{Namespace: "harbor.example", ID: rec.Action.ID})
		if attempt, decodeErr := corpus.Decode[model.ActionRecord](prior); err != nil || decodeErr != nil || attempt.Disposition != model.DispositionUnknown {
			t.Fatalf("the unknown attempt was rewritten: %v %v", err, decodeErr)
		}
		if replayed := h.invoker.Invoke(ctx, reserve()); replayed.Status != capability.StatusExecuted || h.transport.calls != 1 {
			t.Fatalf("replay got %s after %d transport calls", replayed.Status, h.transport.calls)
		}
	})
	t.Run("an unavailable or outcome-less observation leaves the attempt unknown", func(t *testing.T) {
		h, rec := timedOut(t)
		if res := h.invoker.Reconcile(ctx, rec); res.Status != capability.StatusUnknown || res.Requirement != "CHR-EVID-012" || res.LedgerFence == "" {
			t.Fatalf("no outcome named: got %s", res.Status)
		}
		h.observer.outcome, h.observer.err = "reserved", errors.New("read timed out")
		if res := h.invoker.Reconcile(ctx, rec); res.Status != capability.StatusUnknown || h.transport.calls != 1 {
			t.Fatalf("observation failed: got %s", res.Status)
		}
	})
	t.Run("an absent effect fails the key unless the contract declares the retry safe", func(t *testing.T) {
		h, rec := timedOut(t)
		h.observer.result = model.PostconditionViolated
		if res := h.invoker.Reconcile(ctx, rec); res.Status != capability.StatusFailed || res.Requirement != "CHR-CAP-010" {
			t.Fatalf("got %s %s", res.Status, res.Requirement)
		}
		if again := h.invoker.Invoke(ctx, reserve()); again.Status != capability.StatusFailed || h.transport.calls != 1 {
			t.Fatalf("retry got %s after %d transport calls", again.Status, h.transport.calls)
		}
		h, rec = timedOut(t)
		h.amend(func(c *model.CapabilityContract) { c.Idempotency.RetryWhenEffectAbsent = true })
		h.observer.result = model.PostconditionViolated
		h.invoker.Reconcile(ctx, rec)
		h.observer.result = model.PostconditionSatisfied
		if again := h.invoker.Invoke(ctx, reserve()); again.Status != capability.StatusExecuted || h.transport.calls != 2 {
			t.Fatalf("declared-safe retry got %s after %d transport calls", again.Status, h.transport.calls)
		}
	})
	t.Run("a stale fence or a key never attempted cannot be reconciled", func(t *testing.T) {
		h, rec := timedOut(t)
		h.observer.outcome = "reserved"
		stale := rec
		stale.Fence = "not-the-claim"
		if res := h.invoker.Reconcile(ctx, stale); res.Status != capability.StatusDenied || !errors.Is(res.Err, capability.ErrLedgerFence) || h.observer.calls != 0 {
			t.Fatalf("stale fence got %s err %v after %d observations", res.Status, res.Err, h.observer.calls)
		}
		fresh := rec
		fresh.Invocation.IdempotencyKey = "IDEMP-NEVER-SENT"
		if res := h.invoker.Reconcile(ctx, fresh); res.Status != capability.StatusDenied || res.Requirement != "CHR-EVID-012" {
			t.Fatalf("unattempted key got %s", res.Status)
		}
		if res := h.invoker.Invoke(ctx, fresh.Invocation); res.Status != capability.StatusExecuted {
			t.Fatalf("the probed key was left claimed: %s (%s)", res.Status, res.Reason)
		}
	})
}
