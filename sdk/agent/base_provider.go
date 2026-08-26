package agent

import (
	"context"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// BaseProvider serves agent documents from any corpus.Source.
type BaseProvider struct {
	corpus.AbstractDocumentProvider
}

var _ Provider = (*BaseProvider)(nil)

func NewBaseProvider(s corpus.Source) *BaseProvider {
	return &BaseProvider{corpus.AbstractDocumentProvider{Source: s}}
}

func (p *BaseProvider) Identity(ctx context.Context, owner string, ref model.Ref) (model.AgentIdentity, error) {
	return corpus.ResolveAs[model.AgentIdentity](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindAgentIdentity)
}

func (p *BaseProvider) Definition(ctx context.Context, owner string, ref model.Ref) (model.AgentDefinition, error) {
	return corpus.ResolveAs[model.AgentDefinition](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindAgentDefinition)
}

func (p *BaseProvider) Runtime(ctx context.Context, owner string, ref model.Ref) (model.AgentRuntime, error) {
	return corpus.ResolveAs[model.AgentRuntime](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindAgentRuntime)
}

func (p *BaseProvider) Definitions(ctx context.Context) ([]model.AgentDefinition, error) {
	return corpus.ListAs[model.AgentDefinition](ctx, &p.AbstractDocumentProvider, model.KindAgentDefinition)
}

func (p *BaseProvider) Runtimes(ctx context.Context) ([]model.AgentRuntime, error) {
	return corpus.ListAs[model.AgentRuntime](ctx, &p.AbstractDocumentProvider, model.KindAgentRuntime)
}
