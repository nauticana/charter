package authority

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/model"
)

func TestLimitEvaluator(t *testing.T) {
	two := 2
	zero := 0
	var e LimitEvaluator
	cases := []struct {
		name     string
		limits   []model.Limit
		measures map[string]Measure
		wantFail string
	}{
		{"no limits", nil, nil, ""},
		{"minor units within", []model.Limit{{LimitKind: "v", Operator: "lte", Value: json.Number("2500000"), Currency: "USD", CurrencyExponent: &two}},
			map[string]Measure{"v": {Value: int64(2500000), Currency: "USD", CurrencyExponent: 2}}, ""},
		{"minor units over", []model.Limit{{LimitKind: "v", Operator: "lt", Value: float64(100), Currency: "JPY", CurrencyExponent: &zero}},
			map[string]Measure{"v": {Value: 100, Currency: "JPY", CurrencyExponent: 0}}, "not lt"},
		{"exponent differs", []model.Limit{{LimitKind: "v", Operator: "lte", Value: 100, Currency: "USD", CurrencyExponent: &two}},
			map[string]Measure{"v": {Value: 100, Currency: "USD", CurrencyExponent: 0}}, "exponent"},
		{"fractional minor units rejected", []model.Limit{{LimitKind: "v", Operator: "lte", Value: 100.5, Currency: "USD", CurrencyExponent: &two}},
			map[string]Measure{"v": {Value: 100, Currency: "USD", CurrencyExponent: 2}}, "integer minor units"},
		{"unit differs", []model.Limit{{LimitKind: "d", Operator: "lte", Value: 72, Unit: "hour"}}, map[string]Measure{"d": {Value: 2, Unit: "day"}}, "unit"},
		{"numeric string measure", []model.Limit{{LimitKind: "d", Operator: "gte", Value: 1, Unit: "hour"}}, map[string]Measure{"d": {Value: "48", Unit: "hour"}}, ""},
		{"non-numeric measure", []model.Limit{{LimitKind: "d", Operator: "gte", Value: 1}}, map[string]Measure{"d": {Value: "many"}}, "not numeric"},
		{"boolean eq", []model.Limit{{LimitKind: "b", Operator: "eq", Value: true}}, map[string]Measure{"b": {Value: true}}, ""},
		{"boolean gt unsupported", []model.Limit{{LimitKind: "b", Operator: "gt", Value: true}}, map[string]Measure{"b": {Value: true}}, "only with eq"},
		{"unknown operator", []model.Limit{{LimitKind: "d", Operator: "between", Value: 1}}, map[string]Measure{"d": {Value: 1}}, "not between"},
	}
	for _, tc := range cases {
		reason := e.Check(tc.limits, tc.measures)
		if tc.wantFail == "" && reason != "" {
			t.Errorf("%s: unexpected failure %q", tc.name, reason)
		}
		if tc.wantFail != "" && !strings.Contains(reason, tc.wantFail) {
			t.Errorf("%s: got %q, want failure mentioning %q", tc.name, reason, tc.wantFail)
		}
	}
}

func TestDelegationPolicy(t *testing.T) {
	var g model.AuthorityGrant
	if (DelegationPolicy{}).RemainingDepth(g) != 0 {
		t.Error("direct grant permits delegation")
	}
	g.Delegation = &model.Delegation{MaxRedelegationDepth: 2}
	if (DelegationPolicy{}).RemainingDepth(g) != 2 {
		t.Error("delegation depth not reported")
	}

	at := time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)
	validity := &model.Validity{From: "2026-01-01", To: "2026-12-31"}
	chain := model.AuthorityChain{
		{GrantRef: model.Ref{ID: "GRANT-INNER"}, Delegator: model.ObjectRef{Kind: model.KindAgentIdentity, ID: "AGENT-PARENT"}, RemainingDepth: 0, Validity: validity},
		{GrantRef: model.Ref{ID: "GRANT-OUTER"}, Delegator: model.ObjectRef{Kind: model.KindHumanIdentity, ID: "HUMAN-OWNER"}, RemainingDepth: 1, Validity: validity, ApprovalRequired: true},
	}
	policy := DelegationPolicy{}
	if reason := policy.Check(chain, at, nil); reason != "" {
		t.Errorf("valid chain: %q", reason)
	}
	if !policy.ApprovalRequired(chain) {
		t.Error("delegated approval bound ignored")
	}
	invalidDepth := append(model.AuthorityChain(nil), chain...)
	invalidDepth[0].RemainingDepth = 1
	if reason := policy.Check(invalidDepth, at, nil); !strings.Contains(reason, "remaining depth") {
		t.Errorf("widened depth accepted: %q", reason)
	}
	expired := append(model.AuthorityChain(nil), chain...)
	expired[0].Validity = &model.Validity{From: "2025-01-01", To: "2025-12-31"}
	if reason := policy.Check(expired, at, nil); !strings.Contains(reason, "not effective") {
		t.Errorf("expired hop accepted: %q", reason)
	}
	two := 2
	bounded := append(model.AuthorityChain(nil), chain...)
	bounded[0].Limits = []model.Limit{{LimitKind: "delegated-budget", Operator: "lte", Value: int64(1000), Currency: "EUR", CurrencyExponent: &two}}
	over := map[string]Measure{"delegated-budget": {Value: int64(1001), Currency: "EUR", CurrencyExponent: 2}}
	if reason := policy.Check(bounded, at, over); !strings.Contains(reason, "delegated-budget") {
		t.Errorf("delegated budget exceeded without refusal: %q", reason)
	}
}
