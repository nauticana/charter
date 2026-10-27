package predicate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"cel.dev/cel-go/cel"

	"github.com/nauticana/charter/sdk/model"
)

// DefaultCELCostLimit bounds the evaluation cost of one expression when CEL.CostLimit is zero.
const DefaultCELCostLimit = 100_000

// CEL evaluates Common Expression Language predicates with the variables input and facts; numbers compare across
// integer and floating-point types. The zero value is ready to use.
type CEL struct {
	CostLimit uint64

	once sync.Once
	env  *cel.Env
	err  error
}

var _ Evaluator = (*CEL)(nil)

func (c *CEL) Evaluate(ctx context.Context, p model.Predicate, input any, facts map[string]any) (bool, error) {
	if p.Language != model.PredicateCEL {
		return false, fmt.Errorf("%w: %q", ErrUnsupportedLanguage, p.Language)
	}
	c.once.Do(func() {
		c.env, c.err = cel.NewEnv(cel.Variable("input", cel.DynType), cel.Variable("facts", cel.MapType(cel.StringType, cel.DynType)),
			cel.CrossTypeNumericComparisons(true))
	})
	if c.err != nil {
		return false, c.err
	}
	ast, issues := c.env.Compile(p.Expression)
	if issues != nil && issues.Err() != nil {
		return false, issues.Err()
	}
	if ast.OutputType() != cel.BoolType && ast.OutputType() != cel.DynType {
		return false, fmt.Errorf("predicate yields %s, not bool", ast.OutputType())
	}
	limit := c.CostLimit
	if limit == 0 {
		limit = DefaultCELCostLimit
	}
	prg, err := c.env.Program(ast, cel.CostLimit(limit), cel.InterruptCheckFrequency(100))
	if err != nil {
		return false, err
	}
	in, err := plain(input)
	if err != nil {
		return false, err
	}
	known, err := plain(facts)
	if err != nil {
		return false, err
	}
	if known == nil {
		known = map[string]any{}
	}
	out, _, err := prg.ContextEval(ctx, map[string]any{"input": in, "facts": known})
	if err != nil {
		return false, err
	}
	b, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("predicate yields %T, not bool", out.Value())
	}
	return b, nil
}

// plain converts a value to JSON data, integers kept exact, so a struct and its JSON form evaluate alike.
func plain(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	return numbers(out), nil
}

func numbers(v any) any {
	switch x := v.(type) {
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i
		}
		f, _ := x.Float64()
		return f
	case map[string]any:
		for k, e := range x {
			x[k] = numbers(e)
		}
	case []any:
		for k, e := range x {
			x[k] = numbers(e)
		}
	}
	return v
}
