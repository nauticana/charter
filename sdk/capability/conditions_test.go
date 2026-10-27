package capability_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/nauticana/charter/sdk/agent"
	"github.com/nauticana/charter/sdk/binding"
	"github.com/nauticana/charter/sdk/capability"
	"github.com/nauticana/charter/sdk/information"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/organization"
	"github.com/nauticana/charter/sdk/predicate"
)

// factObserver reports facts before execution and a result per postcondition after it, each evidenced unless unevidenced.
type factObserver struct {
	calls       int
	facts       map[string]any
	results     map[string][]string
	unevidenced bool
	preErr      error
	postErr     error
}

func (o *factObserver) Observe(_ context.Context, req binding.ObservationRequest) (binding.Observation, error) {
	o.calls++
	obs := binding.Observation{Facts: o.facts}
	if !o.unevidenced {
		obs.EvidenceRecordIDs = []model.Ref{{ID: "EVR-0042-STOCK-CORRECTED"}}
	}
	if len(req.Preconditions) > 0 {
		return obs, o.preErr
	}
	for _, p := range req.Postconditions {
		for _, result := range o.results[p.ID] {
			obs.Evaluations = append(obs.Evaluations, model.PostconditionEvaluation{PostconditionID: p.ID, Result: result, Reason: "fact observation",
				EvidenceRecordIDs: obs.EvidenceRecordIDs})
		}
	}
	return obs, o.postErr
}

func (h *harness) observeFacts() *factObserver {
	o := &factObserver{facts: map[string]any{"eligible": 40, "reserved": 38}, results: map[string][]string{"POST-RESERVATION-HELD": {model.PostconditionSatisfied}}}
	h.invoker.Observer = o
	return o
}

func quantity(n int) map[string]any { return map[string]any{"orderId": "ORD-2026-0173", "quantity": n} }

const quantitySchema = `{"type":"object","properties":{"orderId":{"type":"string"},"quantity":{"type":"integer","minimum":1}},"required":["orderId","quantity"]}`

func TestInputsAreValidatedBeforeAuthority(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		schema model.DataSchema
		inputs any
		want   capability.Status
	}{
		"valid":               {model.DataSchema{JSONSchema: json.RawMessage(quantitySchema)}, quantity(38), capability.StatusExecuted},
		"missing member":      {model.DataSchema{JSONSchema: json.RawMessage(quantitySchema)}, map[string]any{"orderId": "ORD-2026-0173"}, capability.StatusDenied},
		"no inputs":           {model.DataSchema{JSONSchema: json.RawMessage(quantitySchema)}, nil, capability.StatusDenied},
		"wrong type":          {model.DataSchema{JSONSchema: json.RawMessage(quantitySchema)}, quantity(0), capability.StatusDenied},
		"external reference":  {model.DataSchema{JSONSchema: json.RawMessage(`{"$ref":"file:///etc/passwd"}`)}, quantity(38), capability.StatusDenied},
		"shape without shape": {model.DataSchema{InformationID: &model.Ref{ID: "INFO-CREDIT-STATUS-SUMMARY"}}, quantity(38), capability.StatusDenied},
	} {
		h := newHarness(t)
		h.invoker.Definitions = information.NewBaseProvider(h.corpus)
		schema := tc.schema
		h.amend(func(c *model.CapabilityContract) { c.InputSchema = &schema })
		inv := reserve()
		inv.Inputs = tc.inputs
		res := h.invoker.Invoke(ctx, inv)
		if res.Status != tc.want {
			t.Errorf("%s: got %s (%s)", name, res.Status, res.Reason)
			continue
		}
		h.recorded(res)
		if tc.want == capability.StatusDenied && (res.Requirement != "CHR-CAP-011" || res.Authority.Result != "" || h.transport.calls != 0) {
			t.Errorf("%s: denied under %s after authority %q and %d transport calls", name, res.Requirement, res.Authority.Result, h.transport.calls)
		}
	}
}

func TestInputsTypedByAnInformationShape(t *testing.T) {
	h := newHarness(t)
	h.amend(func(c *model.CapabilityContract) {
		c.InputSchema = &model.DataSchema{InformationID: &model.Ref{ID: "INFO-ORDER-EXCEPTION"}}
	})
	inv := reserve()
	inv.Inputs = map[string]any{"orderId": "ORD-2026-0173", "lineId": "10", "reservedQuantity": 38}
	if res := h.invoker.Invoke(context.Background(), inv); res.Status != capability.StatusDenied || !strings.Contains(res.Reason, "no information provider") {
		t.Fatalf("shape without a provider: %s (%s)", res.Status, res.Reason)
	}
	h = newHarness(t)
	h.invoker.Definitions = information.NewBaseProvider(h.corpus)
	h.amend(func(c *model.CapabilityContract) {
		c.InputSchema = &model.DataSchema{InformationID: &model.Ref{ID: "INFO-ORDER-EXCEPTION"}}
	})
	if res := h.invoker.Invoke(context.Background(), inv); res.Status != capability.StatusExecuted {
		t.Fatalf("shaped input: %s (%s)", res.Status, res.Reason)
	}
	inv.Inputs, inv.IdempotencyKey = map[string]any{"orderId": "ORD-2026-0173", "lineId": "10", "note": "undeclared"}, "IDEMP-0042-2"
	if res := h.invoker.Invoke(context.Background(), inv); res.Status != capability.StatusDenied || res.Requirement != "CHR-CAP-011" {
		t.Fatalf("undeclared member: %s (%s)", res.Status, res.Reason)
	}
}

func TestOutputsOutsideTheSchemaAreUnknown(t *testing.T) {
	h := newHarness(t)
	h.amend(func(c *model.CapabilityContract) {
		c.OutputSchema = &model.DataSchema{JSONSchema: json.RawMessage(`{"type":"object","required":["reservation"]}`)}
	})
	h.transport.resp.Outputs = map[string]any{"other": 1}
	res := h.invoker.Invoke(context.Background(), reserve())
	if res.Status != capability.StatusUnknown || res.Requirement != "CHR-CAP-011" || h.observer.calls != 0 {
		t.Fatalf("got %s %s (%s)", res.Status, res.Requirement, res.Reason)
	}
	h.recorded(res)
	if again := h.invoker.Invoke(context.Background(), reserve()); again.Status != capability.StatusUnknown || h.transport.calls != 1 {
		t.Fatalf("retry of an unknown outcome: %s after %d calls", again.Status, h.transport.calls)
	}
}

func TestPredicatePreconditionsGateExecution(t *testing.T) {
	ctx := context.Background()
	stock := model.Precondition{ID: "PRE-STOCK", Statement: "eligible stock covers the quantity",
		Predicate: &model.Predicate{Language: model.PredicateCEL, Expression: "facts.eligible >= input.quantity"}}
	for name, tc := range map[string]struct {
		arrange func(*harness, *factObserver)
		want    capability.Status
		result  string
		req     string
	}{
		"satisfied":           {func(*harness, *factObserver) {}, capability.StatusExecuted, model.PostconditionSatisfied, ""},
		"violated":            {func(_ *harness, o *factObserver) { o.facts["eligible"] = 12 }, capability.StatusDenied, model.PostconditionViolated, ""},
		"fact missing":        {func(_ *harness, o *factObserver) { delete(o.facts, "eligible") }, capability.StatusDenied, model.PostconditionUnknown, ""},
		"unevidenced":         {func(_ *harness, o *factObserver) { o.unevidenced = true }, capability.StatusDenied, model.PostconditionUnknown, ""},
		"unobservable":        {func(_ *harness, o *factObserver) { o.preErr = errors.New("stock read timed out") }, capability.StatusDenied, model.PostconditionUnknown, ""},
		"no evaluator":        {func(h *harness, _ *factObserver) { h.invoker.Predicates = nil }, capability.StatusDenied, "", ""},
		"no observer at all":  {func(h *harness, _ *factObserver) { h.invoker.Observer = nil }, capability.StatusDenied, "", "CHR-CAP-010"},
		"unsupported grammar": {func(h *harness, _ *factObserver) { h.invoker.Predicates = rejecting{} }, capability.StatusDenied, model.PostconditionUnknown, ""},
	} {
		h := newHarness(t)
		o := h.observeFacts()
		h.invoker.Predicates = &predicate.CEL{}
		h.amend(func(c *model.CapabilityContract) { c.Preconditions = append(c.Preconditions, stock) })
		tc.arrange(h, o)
		inv := reserve()
		inv.Inputs = quantity(38)
		res := h.invoker.Invoke(ctx, inv)
		if res.Status != tc.want {
			t.Errorf("%s: got %s (%s)", name, res.Status, res.Reason)
			continue
		}
		rec := h.recorded(res)
		if tc.req == "" {
			tc.req = "CHR-CAP-012"
		}
		if tc.want == capability.StatusDenied && (res.Requirement != tc.req || h.transport.calls != 0) {
			t.Errorf("%s: denied under %s after %d transport calls", name, res.Requirement, h.transport.calls)
		}
		if tc.result != "" && (len(rec.PreconditionEvaluations) != 1 || rec.PreconditionEvaluations[0].PreconditionID != "PRE-STOCK" || rec.PreconditionEvaluations[0].Result != tc.result) {
			t.Errorf("%s: precondition evidence %+v", name, rec.PreconditionEvaluations)
		}
	}
}

func TestPreconditionsLeaveReplayAndRetryIntact(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	o := h.observeFacts()
	h.invoker.Predicates = &predicate.CEL{}
	h.amend(func(c *model.CapabilityContract) {
		c.Preconditions = append(c.Preconditions, model.Precondition{ID: "PRE-STOCK", Statement: "s",
			Predicate: &model.Predicate{Language: model.PredicateCEL, Expression: "facts.eligible >= input.quantity"}})
	})
	inv := reserve()
	inv.Inputs = quantity(38)
	o.facts["eligible"] = 12
	if res := h.invoker.Invoke(ctx, inv); res.Status != capability.StatusDenied || res.LedgerFence != "" {
		t.Fatalf("violated precondition: %s fence %q", res.Status, res.LedgerFence)
	}
	o.facts["eligible"] = 40
	first := h.invoker.Invoke(ctx, inv)
	if first.Status != capability.StatusExecuted || h.transport.calls != 1 {
		t.Fatalf("a denied key must stay usable: %s (%s)", first.Status, first.Reason)
	}
	o.facts["eligible"] = 2
	if replay := h.invoker.Invoke(ctx, inv); replay.Status != capability.StatusExecuted || replay.Action.ID != first.Action.ID || h.transport.calls != 1 {
		t.Fatalf("a replay must return the settled result, not re-check the precondition: %s (%s)", replay.Status, replay.Reason)
	}
}

type rejecting struct{}

func (rejecting) Evaluate(context.Context, model.Predicate, any, map[string]any) (bool, error) {
	return false, predicate.ErrUnsupportedLanguage
}

func TestPostconditionPredicateDecidesTheEffect(t *testing.T) {
	ctx := context.Background()
	held := &model.Predicate{Language: model.PredicateCEL, Expression: "facts.reserved == input.quantity"}
	for name, tc := range map[string]struct {
		reported []string
		reserved int
		want     capability.Status
		cause    string
	}{
		"predicate alone holds":       {nil, 38, capability.StatusExecuted, ""},
		"predicate alone fails":       {nil, 12, capability.StatusFailed, ""},
		"observer agrees":             {[]string{model.PostconditionSatisfied}, 38, capability.StatusExecuted, ""},
		"observer contradicts":        {[]string{model.PostconditionViolated}, 38, capability.StatusUnknown, model.UnknownContradictory},
		"observer contradicts itself": {[]string{model.PostconditionSatisfied, model.PostconditionViolated}, 38, capability.StatusUnknown, model.UnknownContradictory},
	} {
		h := newHarness(t)
		o := h.observeFacts()
		h.invoker.Predicates = &predicate.CEL{}
		h.amend(func(c *model.CapabilityContract) { c.Postconditions[0].Predicate = held })
		o.results["POST-RESERVATION-HELD"], o.facts["reserved"] = tc.reported, tc.reserved
		inv := reserve()
		inv.Inputs = quantity(38)
		res := h.invoker.Invoke(ctx, inv)
		rec := h.recorded(res)
		if res.Status != tc.want || rec.UnknownCause != tc.cause {
			t.Errorf("%s: got %s cause %q (%s)", name, res.Status, rec.UnknownCause, res.Reason)
		}
	}
}

func TestPartialEffectIsEscalatedAndNeverRetried(t *testing.T) {
	ctx := context.Background()
	arrange := func(t *testing.T, retrySafe bool) (*harness, *factObserver) {
		h := newHarness(t)
		o := h.observeFacts()
		o.results["POST-EXPIRY-SET"] = []string{model.PostconditionViolated}
		h.amend(func(c *model.CapabilityContract) {
			c.Idempotency.RetryWhenEffectAbsent = retrySafe
			c.Postconditions = append(c.Postconditions, model.Postcondition{ID: "POST-EXPIRY-SET", Statement: "The reservation carries the requested expiry",
				VerificationRequired: true, Outcomes: []string{"reserved"}, ViolationBusinessError: "invalid order state"})
		})
		return h, o
	}
	h, _ := arrange(t, true)
	h.invoker.Escalation = &capability.BaseEscalator{Recipients: &capability.BaseAccountableRecipient{Agents: agent.NewBaseProvider(h.corpus),
		Organization: organization.NewBaseProvider(h.corpus)}, IDs: h.invoker.IDs, Statuses: []capability.Status{capability.StatusUnknown}}
	res := h.invoker.Invoke(ctx, reserve())
	rec := h.recorded(res)
	if res.Status != capability.StatusFailed || res.Requirement != "CHR-EVID-016" || res.BusinessError != "" || rec.Effect != model.EffectPartial || res.Exception == nil || res.Escalation == nil {
		t.Fatalf("partial effect: %s %s effect %q exception %v (%s)", res.Status, res.Requirement, rec.Effect, res.Exception, res.Reason)
	}
	if again := h.invoker.Invoke(ctx, reserve()); again.Status != capability.StatusFailed || h.transport.calls != 1 {
		t.Fatalf("a partial effect was retried: %s after %d calls", again.Status, h.transport.calls)
	}
	h, _ = arrange(t, false)
	if res := h.invoker.Invoke(ctx, reserve()); res.Status != capability.StatusFailed || res.Err == nil || !strings.Contains(res.Err.Error(), "requires an escalator") {
		t.Fatalf("partial effect without an escalator: %s %v", res.Status, res.Err)
	}
	h, o := arrange(t, false)
	o.results["POST-RESERVATION-HELD"] = []string{model.PostconditionViolated}
	if res := h.invoker.Invoke(ctx, reserve()); res.Status != capability.StatusFailed || res.Requirement != "CHR-CAP-010" || res.Action.Effect != model.EffectAbsent {
		t.Fatalf("absent effect: %s effect %q", res.Status, res.Action.Effect)
	}
}

func TestUnobservedEffectNamesItsCause(t *testing.T) {
	for name, tc := range map[string]struct {
		arrange func(*factObserver)
		cause   string
	}{
		"observer failed":  {func(o *factObserver) { o.postErr = errors.New("read timed out") }, model.UnknownObservationFailed},
		"unevidenced":      {func(o *factObserver) { o.unevidenced = true }, model.UnknownUnevidenced},
		"not evaluated":    {func(o *factObserver) { delete(o.results, "POST-RESERVATION-HELD") }, model.UnknownUnevidenced},
		"conflicting read": {func(o *factObserver) { o.results["POST-RESERVATION-HELD"] = []string{"satisfied", "violated"} }, model.UnknownContradictory},
	} {
		h := newHarness(t)
		tc.arrange(h.observeFacts())
		res := h.invoker.Invoke(context.Background(), reserve())
		if rec := h.recorded(res); res.Status != capability.StatusUnknown || rec.Effect != model.EffectUnobserved || rec.UnknownCause != tc.cause {
			t.Errorf("%s: %s effect %q cause %q", name, res.Status, rec.Effect, rec.UnknownCause)
		}
	}
}

func TestReadWithoutLedgerSurvivesTransportFailure(t *testing.T) {
	h := newHarness(t)
	h.invoker.ReadWithoutGrant, h.invoker.Ledger = true, nil
	h.transport.err = binding.ErrOutcomeUnknown
	read := reserve()
	read.CapabilityID, read.RequiredFeatures, read.IdempotencyKey, read.MaterialInputsDigest = model.Ref{ID: "CAP-READ-ORDER-EXCEPTION"}, nil, "", ""
	if res := h.invoker.Invoke(context.Background(), read); res.Status != capability.StatusUnknown {
		t.Fatalf("read transport failure: %s (%s)", res.Status, res.Reason)
	}
}
