package information

import (
	"context"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// BaseProvider serves information documents from any corpus.Source.
type BaseProvider struct {
	corpus.AbstractDocumentProvider
}

var _ Provider = (*BaseProvider)(nil)

func NewBaseProvider(s corpus.Source) *BaseProvider {
	return &BaseProvider{corpus.AbstractDocumentProvider{Source: s}}
}

func (p *BaseProvider) Definition(ctx context.Context, owner string, ref model.Ref) (model.InformationDefinition, error) {
	return corpus.ResolveAs[model.InformationDefinition](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindInformationDefinition)
}

func (p *BaseProvider) Policy(ctx context.Context, owner string, ref model.Ref) (model.InformationGovernancePolicy, error) {
	return corpus.ResolveAs[model.InformationGovernancePolicy](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindInformationGovernancePolicy)
}
