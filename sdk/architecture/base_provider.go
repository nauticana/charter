package architecture

import (
	"context"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// BaseProvider serves architecture documents from any corpus.Source.
type BaseProvider struct {
	corpus.AbstractDocumentProvider
}

var _ Provider = (*BaseProvider)(nil)

func NewBaseProvider(s corpus.Source) *BaseProvider {
	return &BaseProvider{corpus.AbstractDocumentProvider{Source: s}}
}

func (p *BaseProvider) State(ctx context.Context, owner string, ref model.Ref) (model.ArchitectureState, error) {
	return corpus.ResolveAs[model.ArchitectureState](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindArchitectureState)
}

func (p *BaseProvider) Gap(ctx context.Context, owner string, ref model.Ref) (model.Gap, error) {
	return corpus.ResolveAs[model.Gap](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindGap)
}

func (p *BaseProvider) RoadmapItem(ctx context.Context, owner string, ref model.Ref) (model.RoadmapItem, error) {
	return corpus.ResolveAs[model.RoadmapItem](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindRoadmapItem)
}

func (p *BaseProvider) States(ctx context.Context) ([]model.ArchitectureState, error) {
	return corpus.ListAs[model.ArchitectureState](ctx, &p.AbstractDocumentProvider, model.KindArchitectureState)
}

func (p *BaseProvider) Gaps(ctx context.Context) ([]model.Gap, error) {
	return corpus.ListAs[model.Gap](ctx, &p.AbstractDocumentProvider, model.KindGap)
}

func (p *BaseProvider) RoadmapItems(ctx context.Context) ([]model.RoadmapItem, error) {
	return corpus.ListAs[model.RoadmapItem](ctx, &p.AbstractDocumentProvider, model.KindRoadmapItem)
}
