package predicate

import (
	"context"
	"errors"
	"testing"

	"github.com/nauticana/charter/sdk/model"
)

func TestCEL(t *testing.T) {
	c := &CEL{}
	ctx := context.Background()
	cel := func(expr string) model.Predicate {
		return model.Predicate{Language: model.PredicateCEL, Expression: expr}
	}
	input := struct {
		Quantity int `json:"quantity"`
	}{38}
	for name, tc := range map[string]struct {
		expr  string
		facts map[string]any
		want  bool
	}{
		"satisfied across integer and double": {"facts.eligible >= input.quantity", map[string]any{"eligible": 40.0}, true},
		"violated":                            {"facts.eligible >= input.quantity", map[string]any{"eligible": 12}, false},
	} {
		if got, err := c.Evaluate(ctx, cel(tc.expr), input, tc.facts); err != nil || got != tc.want {
			t.Errorf("%s: %v %v, want %v", name, got, err, tc.want)
		}
	}
	for name, tc := range map[string]struct {
		p     model.Predicate
		facts map[string]any
	}{
		"missing fact":         {cel("facts.eligible >= input.quantity"), nil},
		"not a boolean":        {cel("input.quantity + 1"), nil},
		"does not compile":     {cel("input.quantity >="), nil},
		"unsupported language": {model.Predicate{Language: "rego", Expression: "true"}, nil},
		"over the cost limit":  {cel("facts.items.all(x, facts.items.all(y, x + y >= 0))"), map[string]any{"items": make([]int, 1000)}},
	} {
		if got, err := c.Evaluate(ctx, tc.p, input, tc.facts); err == nil {
			t.Errorf("%s: decided %v, want an error", name, got)
		}
	}
	if _, err := c.Evaluate(ctx, model.Predicate{Language: "rego"}, nil, nil); !errors.Is(err, ErrUnsupportedLanguage) {
		t.Errorf("unsupported language: %v", err)
	}
}
