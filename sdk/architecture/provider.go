// Package architecture resolves architecture states, gaps, and roadmap items and checks their consistency
// without prescribing an enterprise-architecture method.
package architecture

import (
	"context"

	"github.com/nauticana/charter/sdk/model"
)

type Provider interface {
	State(ctx context.Context, ownerNamespace string, ref model.Ref) (model.ArchitectureState, error)
	Gap(ctx context.Context, ownerNamespace string, ref model.Ref) (model.Gap, error)
	RoadmapItem(ctx context.Context, ownerNamespace string, ref model.Ref) (model.RoadmapItem, error)
	States(ctx context.Context) ([]model.ArchitectureState, error)
	Gaps(ctx context.Context) ([]model.Gap, error)
	RoadmapItems(ctx context.Context) ([]model.RoadmapItem, error)
}
