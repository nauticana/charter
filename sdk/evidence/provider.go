package evidence

import (
	"context"

	"github.com/nauticana/charter/sdk/model"
)

type Provider interface {
	Action(ctx context.Context, ownerNamespace string, ref model.Ref) (model.ActionRecord, error)
	Record(ctx context.Context, ownerNamespace string, ref model.Ref) (model.EvidenceRecord, error)
	Exception(ctx context.Context, ownerNamespace string, ref model.Ref) (model.ExceptionRecord, error)
	Escalation(ctx context.Context, ownerNamespace string, ref model.Ref) (model.Escalation, error)
	Bundle(ctx context.Context, ownerNamespace string, ref model.Ref) (model.EvidenceBundle, error)
	Actions(ctx context.Context) ([]model.ActionRecord, error)
	Records(ctx context.Context) ([]model.EvidenceRecord, error)
	Exceptions(ctx context.Context) ([]model.ExceptionRecord, error)
	Escalations(ctx context.Context) ([]model.Escalation, error)
}
