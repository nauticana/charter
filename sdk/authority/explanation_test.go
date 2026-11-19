package authority

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/identity"
	"github.com/nauticana/charter/sdk/model"
)

func TestExplainReportsWhatDecided(t *testing.T) {
	g := harborGrant(t)
	ev := &AbstractEvaluator{Source: &BaseGrantSource{Items: []model.AuthorityGrant{g}}}
	req := Request{Namespace: g.Namespace, EnterpriseID: *g.EnterpriseID, Actor: g.Actor, CapabilityID: g.CapabilityID, ResourceScope: g.ResourceScope,
		OrganizationalContext: g.OrganizationalContext, At: time.Date(2026, 6, 18, 17, 0, 0, 0, time.UTC),
		Measures: map[string]Measure{"reservation-value": {Value: int64(100), Currency: "USD", CurrencyExponent: 2}, "reservation-duration": {Value: 1, Unit: "hour"}}}
	x := ev.Explain(context.Background(), req)
	if x.Decision != Allowed || len(x.Basis) != 1 || x.Basis[0].Result != Satisfied || x.Basis[0].Ref.ID != g.ID || len(x.Missing) != 0 {
		t.Fatalf("allowed: %+v", x)
	}
	req.At = time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	x = ev.Explain(context.Background(), req)
	if d := ev.Evaluate(context.Background(), req); x.Decision != d.Result || x.Decision != Denied || x.Basis[0].Result != Unsatisfied || len(x.Missing) != 1 {
		t.Fatalf("explanation diverges from evaluation: %+v %+v", x, d)
	}
	hidden := x.Redact(func(model.ObjectRef) bool { return false })
	if b := hidden.Basis[0]; b.Ref != nil || !b.Redacted || b.Reason != "redacted" || b.Result != Unsatisfied || x.Basis[0].Ref == nil {
		t.Errorf("redaction: %+v, original %+v", b, x.Basis[0])
	}
	req.CapabilityID = model.Ref{ID: "CAP-OTHER"}
	if x := ev.Explain(context.Background(), req); x.Decision != Missing || len(x.Basis) != 0 || len(x.Missing) != 1 {
		t.Errorf("missing: %+v", x)
	}
}

func TestGrantLapses(t *testing.T) {
	g := harborGrant(t)
	at := time.Date(2026, 6, 18, 9, 0, 0, 0, time.UTC)
	if reason := Lapse(g, at); reason != "" {
		t.Fatalf("grant in force: %s", reason)
	}
	revoked := at.Add(time.Hour)
	g.RevokedAt = &revoked
	if reason := Lapse(g, at); reason != "" {
		t.Errorf("before revocation: %s", reason)
	}
	if reason := Lapse(g, revoked); !strings.HasPrefix(reason, "revoked at") {
		t.Errorf("at revocation: %q", reason)
	}
	g.RevokedAt = nil
	g.Certification.CertifiedBy = g.Actor
	if Lapse(g, at) == "" {
		t.Error("self-certification accepted")
	}
	g.Validity = nil
	if !strings.HasPrefix(Lapse(g, at), "not effective") {
		t.Error("grant without validity in force")
	}
}

func TestExcuseRelievesOnlyTheNamedConstraint(t *testing.T) {
	ns := "test.example"
	sod := func(id string) model.SodConstraint {
		c := model.SodConstraint{}
		c.Namespace, c.ID = ns, id
		return c
	}
	actor := model.ObjectRef{Kind: model.KindHumanIdentity, ID: "H-1"}
	revoked := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	e := model.SodException{SodConstraintID: model.Ref{ID: "SOD-1"}, Beneficiary: actor, Grantor: model.ObjectRef{Kind: model.KindHumanIdentity, ID: "H-2"}, RevokedAt: &revoked}
	e.Namespace, e.ID, e.Validity = ns, "SODEX-1", &model.Validity{From: "2026-01-01", To: "2026-12-31"}
	at := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	constraints := []model.SodConstraint{sod("SOD-1"), sod("SOD-2")}
	left, err := Excuse(constraints, e, ns, actor, at)
	if err != nil || len(left) != 1 || left[0].ID != "SOD-2" || constraints[0].ID != "SOD-1" {
		t.Fatalf("excuse: %+v %v", left, err)
	}
	for name, f := range map[string]func(*model.SodException, *time.Time){
		"other actor":  func(x *model.SodException, _ *time.Time) { x.Beneficiary.ID = "H-3" },
		"self-granted": func(x *model.SodException, _ *time.Time) { x.Grantor = actor },
		"expired": func(x *model.SodException, _ *time.Time) {
			x.Validity = &model.Validity{From: "2026-01-01", To: "2026-05-31"}
		},
		"revoked":   func(_ *model.SodException, at *time.Time) { *at = revoked },
		"unrelated": func(x *model.SodException, _ *time.Time) { x.SodConstraintID = model.Ref{ID: "SOD-9"} },
	} {
		x, when := e, at
		f(&x, &when)
		if _, err := Excuse(constraints, x, ns, actor, when); err == nil {
			t.Errorf("%s: exception accepted", name)
		}
	}
}

func TestWidens(t *testing.T) {
	two := 2
	usd := func(op string, v int64) model.Limit {
		return model.Limit{LimitKind: "spend", Operator: op, Value: v, Currency: "USD", CurrencyExponent: &two}
	}
	var e LimitEvaluator
	for _, tc := range []struct {
		name         string
		outer, inner model.Limit
		want         bool
	}{
		{"tighter upper", usd("lte", 100), usd("lt", 100), false},
		{"looser upper", usd("lt", 100), usd("lte", 100), true},
		{"higher upper", usd("lte", 100), usd("lte", 101), true},
		{"other direction combines", usd("lte", 100), usd("gte", 500), false},
		{"looser lower", usd("gte", 10), usd("gt", 5), true},
		{"eq within", usd("lte", 100), usd("eq", 50), false},
		{"other currency", usd("lte", 100), model.Limit{LimitKind: "spend", Operator: "lte", Value: int64(50), Currency: "EUR", CurrencyExponent: &two}, true},
		{"other kind", usd("lte", 100), model.Limit{LimitKind: "steps", Operator: "lte", Value: 500}, false},
	} {
		if got := e.Widens(tc.outer, tc.inner); got != tc.want {
			t.Errorf("%s: widens %t, want %t", tc.name, got, tc.want)
		}
	}
}

func TestExplainNamesResolvers(t *testing.T) {
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	g := harborGrant(t)
	g.Delegation = &model.Delegation{Delegator: model.ObjectRef{Kind: model.KindHumanIdentity, ID: "HUM-DELEGATOR"}}
	ev := &AbstractEvaluator{Source: &BaseGrantSource{Items: []model.AuthorityGrant{g, g}}, Identities: identity.NewBaseProvider(c)}
	req := Request{Namespace: g.Namespace, EnterpriseID: *g.EnterpriseID, Actor: g.Actor, CapabilityID: g.CapabilityID, ResourceScope: g.ResourceScope,
		OrganizationalContext: g.OrganizationalContext, At: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)}
	x := ev.Explain(context.Background(), req)
	want := []model.ObjectRef{{Kind: model.KindHumanIdentity, Namespace: g.Namespace, ID: "HUM-DELEGATOR"},
		{Kind: model.KindPosition, Namespace: g.Namespace, ID: "POS-SALES-OPERATIONS-MANAGER"}}
	if len(x.Missing) != 1 || !slices.Equal(x.Missing[0].Resolvers, want) {
		t.Fatalf("resolvers: %+v", x.Missing)
	}
	hidden := x.Redact(func(r model.ObjectRef) bool { return r.Kind == model.KindPosition })
	if !slices.Equal(hidden.Missing[0].Resolvers, want[1:]) || len(x.Missing[0].Resolvers) != 2 {
		t.Errorf("redacted resolvers: %+v, original %+v", hidden.Missing, x.Missing)
	}
	ev.Identities = nil
	req.CapabilityID = model.Ref{ID: "CAP-OTHER"}
	if x := ev.Explain(context.Background(), req); x.Decision != Missing || len(x.Missing[0].Resolvers) != 0 {
		t.Errorf("no grant, no identities: %+v", x.Missing)
	}
}
