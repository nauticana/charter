package authority

import "context"

type Evaluator interface {
	Evaluate(ctx context.Context, req Request) Decision
}
