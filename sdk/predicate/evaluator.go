// Package predicate evaluates the machine-checkable conditions of capability contracts (CHR-CAP-012).
package predicate

import (
	"context"
	"errors"

	"github.com/nauticana/charter/sdk/model"
)

var ErrUnsupportedLanguage = errors.New("unsupported predicate language")

// Evaluator decides a predicate over the invocation input and observed facts. An error means the predicate could not
// be decided, which a caller treats as unknown, never as satisfied.
type Evaluator interface {
	Evaluate(ctx context.Context, p model.Predicate, input any, facts map[string]any) (bool, error)
}
