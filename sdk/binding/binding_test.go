package binding

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

type fakeVendor struct {
	callErr error
	outcome string
}

func (f fakeVendor) MapRequest(_ context.Context, req Request) (any, error) { return req.Inputs, nil }
func (f fakeVendor) Call(context.Context, any) (any, error)                 { return f.outcome, f.callErr }
func (f fakeVendor) MapResponse(_ context.Context, r any) (Response, error) {
	return Response{Outcome: r.(string), ExternalReference: "0000088421"}, nil
}
func (f fakeVendor) MapError(err error) (string, bool) {
	if err.Error() == "sap: no stock" {
		return "insufficient eligible stock", true
	}
	return "", false
}

func TestHarborBinding(t *testing.T) {
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ns := "harbor.example"
	lookup := Lookup{Provider: NewBaseProvider(c)}
	capability := model.Ref{ID: "CAP-RESERVE-ORDER-STOCK"}
	doc, err := lookup.CapabilityBindingFor(ctx, ns, capability, nil)
	if err != nil || doc.ID != "BIND-S4-RESERVE-ORDER-STOCK-1" {
		t.Fatalf("lookup: %s %v", doc.ID, err)
	}
	if _, err := lookup.CapabilityBindingFor(ctx, ns, capability, &model.Ref{ID: "SYSPROFILE-OTHER"}); !errors.Is(err, ErrNoBinding) {
		t.Errorf("other profile: %v", err)
	}
	if ab, err := lookup.AuthorityBindingFor(ctx, ns, capability, nil); err != nil || ab.UnpreservableConstraintBehavior != model.FailClosed {
		t.Errorf("authority binding: %+v %v", ab, err)
	}
	features := Features(doc.FeatureSupport)
	if err := features.Require("reservation-create"); err != nil {
		t.Error(err)
	}
	if err := features.Require("reservation-create", "reservation-expiry", "native-order-lock"); !errors.Is(err, ErrUnsupportedFeature) {
		t.Errorf("unsupported and undeclared features accepted: %v", err)
	}
	evaluated := []model.FeatureSupport{{Feature: "reservation-create", Support: model.SupportSupported}}
	if got := Compare(doc.FeatureSupport, evaluated); got != model.ResultConforming {
		t.Errorf("compare: %s", got)
	}
	if got := Compare(doc.FeatureSupport, []model.FeatureSupport{{Feature: "reservation-create", Support: model.SupportUnsupported}}); got != model.ResultNonconforming {
		t.Errorf("compare unsupported: %s", got)
	}
	if got := Compare(doc.FeatureSupport, nil); got != model.ResultPartial {
		t.Errorf("compare unevaluated: %s", got)
	}
	conf := Conformance(doc.Envelope, doc.FeatureSupport, evaluated, model.ConformanceEnvelope{ConformanceProfile: "system-adapter", ImplementationVersion: "0.1", TestedRuleIDs: []string{"CHR-RULE-BIND-001"}, VerificationTypes: []string{model.VerificationBehavioral}, ResultDate: "2026-06-18"})
	if conf.Result != model.ResultConforming || conf.BindingID.ID != doc.ID || conf.Kind != model.KindBindingConformance {
		t.Errorf("conformance document: %+v", conf)
	}

	req := Request{Capability: capability, RequiredFeatures: []string{"reservation-create"}, Inputs: "order"}
	b := &AbstractBinding{Document: doc, Vendor: fakeVendor{outcome: "reserved"}}
	if resp, err := b.Execute(ctx, req); err != nil || resp.Outcome != "reserved" || resp.ExternalReference == "" {
		t.Errorf("execute: %+v %v", resp, err)
	}
	if _, err := b.Execute(ctx, Request{Capability: capability, RequiredFeatures: []string{"reservation-expiry"}}); !errors.Is(err, ErrNotExecuted) {
		t.Errorf("unsupported feature executed: %v", err)
	}
	if _, err := b.Execute(ctx, Request{Capability: model.Ref{ID: "CAP-OTHER"}}); !errors.Is(err, ErrNotExecuted) {
		t.Errorf("other capability executed: %v", err)
	}
	b.Vendor = fakeVendor{callErr: errors.New("sap: no stock")}
	if resp, err := b.Execute(ctx, req); err != nil || resp.BusinessError != "insufficient eligible stock" {
		t.Errorf("business error mapping: %+v %v", resp, err)
	}
	b.Vendor = fakeVendor{callErr: errors.New("gateway timeout")}
	if _, err := b.Execute(ctx, req); !errors.Is(err, ErrOutcomeUnknown) {
		t.Errorf("unmapped call failure must be unknown: %v", err)
	}
	b.Vendor = fakeVendor{callErr: fmt.Errorf("%w: connection refused", ErrNotExecuted)}
	if _, err := b.Execute(ctx, req); !errors.Is(err, ErrNotExecuted) || errors.Is(err, ErrOutcomeUnknown) {
		t.Errorf("proven non-execution must stay not-executed: %v", err)
	}
	if len(b.Lossy()) != 1 {
		t.Error("lossy mappings not exposed")
	}
}

type recordingHandler struct {
	deliveries []Delivery
	err        error
}

func (h *recordingHandler) Handle(_ context.Context, d Delivery) error {
	h.deliveries = append(h.deliveries, d)
	return h.err
}

func TestEventConsumerAppliesDeliverySemantics(t *testing.T) {
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		`{"charterSpecVersion":"1.0.0","namespace":"harbor.example","kind":"EventBinding","id":"EVTBIND-ONCE","bindingVersion":"1","systemProfileId":"SYSPROFILE-HARBOR-S4-2602","externalEvent":"e","charterTrigger":"scheduled review of an open exception","featureSupport":[{"feature":"event-delivery","support":"supported"}],"deliverySemantics":"at-most-once"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"harbor.example","kind":"EventBinding","id":"EVTBIND-EXACTLY","bindingVersion":"1","systemProfileId":"SYSPROFILE-HARBOR-S4-2602","externalEvent":"e","charterTrigger":"scheduled review of an open exception","featureSupport":[{"feature":"event-delivery","support":"supported"}],"deliverySemantics":"exactly-once"}`,
	} {
		d, err := (corpus.Parser{}).Parse([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if err := c.Add(d); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	at := time.Date(2026, 6, 18, 17, 0, 0, 0, time.UTC)
	handler := &recordingHandler{}
	consumer := &AbstractEventConsumer{Bindings: NewBaseProvider(c), Ledger: NewBaseMemoryDeliveryLedger(), Handler: handler}
	blocked := model.Ref{ID: "EVTBIND-S4-ORDER-BLOCKED-1"}
	if r, err := consumer.Deliver(ctx, "harbor.example", blocked, "evt-1", "payload", at); err != nil || r.Status != DeliveryAccepted || len(handler.deliveries) != 1 || handler.deliveries[0].Trigger != "order-blocked event mapped to a known order" {
		t.Fatalf("first delivery: %+v %v", r, err)
	}
	if r, _ := consumer.Deliver(ctx, "harbor.example", blocked, "evt-1", "payload", at); r.Status != DeliveryDuplicate || len(handler.deliveries) != 1 {
		t.Errorf("redelivery: %+v", r)
	}
	if r, _ := consumer.Deliver(ctx, "harbor.example", blocked, "", "payload", at); r.Status != DeliveryRejected {
		t.Errorf("at-least-once without event id: %+v", r)
	}
	handler.err = errors.New("runtime busy")
	if r, err := consumer.Deliver(ctx, "harbor.example", blocked, "evt-2", "payload", at); err == nil || r.Status != DeliveryRejected {
		t.Errorf("handler failure: %+v %v", r, err)
	}
	handler.err = nil
	if r, _ := consumer.Deliver(ctx, "harbor.example", blocked, "evt-2", "payload", at); r.Status != DeliveryAccepted {
		t.Errorf("redelivery after failure must be handled: %+v", r)
	}
	once := &AbstractEventConsumer{Bindings: NewBaseProvider(c), Ledger: NewBaseMemoryDeliveryLedger(), Handler: handler}
	if r, _ := once.Deliver(ctx, "harbor.example", model.Ref{ID: "EVTBIND-ONCE"}, "evt-3", nil, at); r.Status != DeliveryAccepted {
		t.Errorf("first at-most-once delivery: %+v", r)
	}
	if r, _ := once.Deliver(ctx, "harbor.example", model.Ref{ID: "EVTBIND-ONCE"}, "evt-3", nil, at); r.Status != DeliveryDuplicate {
		t.Errorf("duplicate at-most-once delivery: %+v", r)
	}
	if r, _ := once.Deliver(ctx, "harbor.example", model.Ref{ID: "EVTBIND-EXACTLY"}, "evt-4", nil, at); r.Status != DeliveryRejected || !strings.Contains(r.Reason, "atomic") {
		t.Errorf("unimplementable exactly-once delivery: %+v", r)
	}
	if r, _ := consumer.Deliver(ctx, "harbor.example", model.Ref{ID: "EVTBIND-NOWHERE"}, "evt-4", nil, at); r.Status != DeliveryRejected {
		t.Errorf("unknown binding: %+v", r)
	}
}
