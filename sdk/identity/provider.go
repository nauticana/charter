package identity

import (
	"context"

	"github.com/nauticana/charter/sdk/model"
)

type Provider interface {
	Human(ctx context.Context, ownerNamespace string, ref model.Ref) (model.HumanIdentity, error)
	Agent(ctx context.Context, ownerNamespace string, ref model.Ref) (model.AgentIdentity, error)
}
