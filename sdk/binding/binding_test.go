package binding

import (
	"context"
	"errors"
	"fmt"
	"testing"

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
