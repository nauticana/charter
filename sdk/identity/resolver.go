package identity

import (
	"context"
	"time"

	"github.com/nauticana/charter/sdk/model"
)

// Resolver turns identity references into actors and answers lifecycle-at-time questions.
type Resolver interface {
	Resolve(ctx context.Context, ownerNamespace string, ref model.ObjectRef) (Actor, error)
	ActiveAt(ctx context.Context, ownerNamespace string, ref model.ObjectRef, at time.Time) (Actor, error)
}
