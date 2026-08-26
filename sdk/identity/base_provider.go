package identity

import (
	"context"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// BaseProvider serves identities from any corpus.Source.
type BaseProvider struct {
	corpus.AbstractDocumentProvider
}

var _ Provider = (*BaseProvider)(nil)

func NewBaseProvider(s corpus.Source) *BaseProvider {
	return &BaseProvider{corpus.AbstractDocumentProvider{Source: s}}
}

func (p *BaseProvider) Human(ctx context.Context, ownerNamespace string, ref model.Ref) (model.HumanIdentity, error) {
	return corpus.ResolveAs[model.HumanIdentity](ctx, &p.AbstractDocumentProvider, ownerNamespace, ref, model.KindHumanIdentity)
}

func (p *BaseProvider) Agent(ctx context.Context, ownerNamespace string, ref model.Ref) (model.AgentIdentity, error) {
	return corpus.ResolveAs[model.AgentIdentity](ctx, &p.AbstractDocumentProvider, ownerNamespace, ref, model.KindAgentIdentity)
}

// BaseResolver resolves actors from any corpus.Source.
type BaseResolver struct {
	AbstractResolver
}

var _ Resolver = (*BaseResolver)(nil)

func NewBaseResolver(s corpus.Source) *BaseResolver {
	return &BaseResolver{AbstractResolver{Provider: NewBaseProvider(s)}}
}
