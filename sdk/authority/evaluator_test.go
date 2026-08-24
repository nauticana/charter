package authority

import (
	"context"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func harborGrant(t *testing.T) model.AuthorityGrant {
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	d, ok := c.Get("harbor.example", "AUTH-OEC-STOCK-RESERVATION-2026")
	if !ok {
		t.Fatal("grant missing")
	}
	g, err := corpus.Decode[model.AuthorityGrant](d)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestEvaluate(t *testing.T) {
	g := harborGrant(t)
	ev := &AbstractEvaluator{Source: &BaseGrantSource{Items: []model.AuthorityGrant{g}}}
	at := time.Date(2026, 6, 18, 17, 12, 30, 0, time.UTC)
	within := map[string]Measure{"reservation-value": {Value: int64(1850000), Currency: "USD", CurrencyExponent: 2}, "reservation-duration": {Value: 48, Unit: "hour"}}
	base := Request{Namespace: g.Namespace, EnterpriseID: *g.EnterpriseID, Actor: g.Actor, CapabilityID: g.CapabilityID,
		ResourceScope: g.ResourceScope, At: at, OrganizationalContext: "OU-SALES-OPERATIONS", Measures: within}

	cases := []struct {
		name string
		req  Request
		want Result
	}{
		{"within limits", base, Allowed},
		{"over value limit", with(base, func(r *Request) {
			r.Measures = map[string]Measure{"reservation-value": {Value: int64(2700000), Currency: "USD", CurrencyExponent: 2}, "reservation-duration": within["reservation-duration"]}
		}), Denied},
		{"wrong currency", with(base, func(r *Request) {
			r.Measures = map[string]Measure{"reservation-value": {Value: int64(10000), Currency: "EUR", CurrencyExponent: 2}, "reservation-duration": within["reservation-duration"]}
		}), Denied},
		{"missing measure", with(base, func(r *Request) { r.Measures = map[string]Measure{"reservation-value": within["reservation-value"]} }), Denied},
		{"expired", with(base, func(r *Request) { r.At = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC) }), Denied},
		{"other context", with(base, func(r *Request) { r.OrganizationalContext = "OU-FINANCE" }), Denied},
		{"missing context", with(base, func(r *Request) { r.OrganizationalContext = "" }), Denied},
		{"other resource scope", with(base, func(r *Request) { r.ResourceScope = "all orders" }), Denied},
		{"no grant", with(base, func(r *Request) { r.CapabilityID = model.Ref{ID: "CAP-OTHER"} }), Missing},
	}
	for _, tc := range cases {
		if d := ev.Evaluate(context.Background(), tc.req); d.Result != tc.want {
			t.Errorf("%s: got %s (%s), want %s", tc.name, d.Result, d.Reason, tc.want)
		} else if tc.want == Allowed && (d.GrantRef == nil || d.GrantRef.ID != g.ID || d.GrantRef.Namespace != g.Namespace) {
			t.Errorf("%s: allowed without grant id", tc.name)
		}
	}
	if (DelegationPolicy{}).RemainingDepth(g) != 0 {
		t.Error("Harbor grant must not permit redelegation")
	}
}

func with(r Request, f func(*Request)) Request { f(&r); return r }

func TestSodConflict(t *testing.T) {
	sod := model.SodConstraint{ConstrainedActions: []model.Ref{{ID: "CAP-PROPOSE"}, {ID: "CAP-APPROVE"}}}
	sod.ID = "SOD-1"
	sod.Namespace = "test.example"
	sod.Scope = "CASE-1"
	propose := SodAction{Namespace: sod.Namespace, Capability: model.Ref{ID: "CAP-PROPOSE"}, Scope: sod.Scope}
	approve := SodAction{Namespace: sod.Namespace, Capability: model.Ref{ID: "CAP-APPROVE"}, Scope: sod.Scope}
	evaluator := SodEvaluator{Scopes: ExactSodScopeMatcher{}}
	if ref, ok, err := evaluator.Conflict([]model.SodConstraint{sod}, []SodAction{propose}, approve); err != nil || !ok || ref.ID != "SOD-1" || ref.Namespace != sod.Namespace {
		t.Error("preparer approving must conflict")
	}
	if _, ok, err := evaluator.Conflict([]model.SodConstraint{sod}, []SodAction{{Namespace: sod.Namespace, Capability: model.Ref{ID: "CAP-READ"}, Scope: sod.Scope}}, approve); err != nil || ok {
		t.Error("unrelated prior action must not conflict")
	}
	propose.Scope = "CASE-2"
	if _, ok, err := evaluator.Conflict([]model.SodConstraint{sod}, []SodAction{propose}, approve); err != nil || ok {
		t.Error("action in another scope must not conflict")
	}
	if _, _, err := (SodEvaluator{}).Conflict([]model.SodConstraint{sod}, []SodAction{propose}, approve); err == nil {
		t.Error("scoped constraint without matcher must fail closed")
	}
}
