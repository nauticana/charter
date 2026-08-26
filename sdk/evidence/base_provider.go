package evidence

import (
	"context"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// BaseProvider reads evidence from any corpus.Source, including a Store.
type BaseProvider struct {
	corpus.AbstractDocumentProvider
}

var _ Provider = (*BaseProvider)(nil)

func NewBaseProvider(s corpus.Source) *BaseProvider {
	return &BaseProvider{corpus.AbstractDocumentProvider{Source: s}}
}

func (p *BaseProvider) Action(ctx context.Context, owner string, ref model.Ref) (model.ActionRecord, error) {
	return corpus.ResolveAs[model.ActionRecord](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindActionRecord)
}

func (p *BaseProvider) Record(ctx context.Context, owner string, ref model.Ref) (model.EvidenceRecord, error) {
	return corpus.ResolveAs[model.EvidenceRecord](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindEvidenceRecord)
}

func (p *BaseProvider) Exception(ctx context.Context, owner string, ref model.Ref) (model.ExceptionRecord, error) {
	return corpus.ResolveAs[model.ExceptionRecord](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindExceptionRecord)
}

func (p *BaseProvider) Escalation(ctx context.Context, owner string, ref model.Ref) (model.Escalation, error) {
	return corpus.ResolveAs[model.Escalation](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindEscalation)
}

func (p *BaseProvider) Bundle(ctx context.Context, owner string, ref model.Ref) (model.EvidenceBundle, error) {
	return corpus.ResolveAs[model.EvidenceBundle](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindEvidenceBundle)
}

func (p *BaseProvider) Actions(ctx context.Context) ([]model.ActionRecord, error) {
	return corpus.ListAs[model.ActionRecord](ctx, &p.AbstractDocumentProvider, model.KindActionRecord)
}

func (p *BaseProvider) Records(ctx context.Context) ([]model.EvidenceRecord, error) {
	return corpus.ListAs[model.EvidenceRecord](ctx, &p.AbstractDocumentProvider, model.KindEvidenceRecord)
}
