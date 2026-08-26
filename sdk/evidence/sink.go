package evidence

import (
	"context"

	"github.com/nauticana/charter/sdk/model"
)

// Sink is the append-only output of governed work.
type Sink interface {
	Action(ctx context.Context, a model.ActionRecord) error
	Record(ctx context.Context, r model.EvidenceRecord) error
	Exception(ctx context.Context, e model.ExceptionRecord) error
	Escalation(ctx context.Context, e model.Escalation) error
	Bundle(ctx context.Context, b model.EvidenceBundle) error
}
