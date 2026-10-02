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

// ActionsIn scans every action of the source, which has no execution-context index.
func (p *BaseProvider) ActionsIn(ctx context.Context, owner, executionContextID string) ([]model.ActionRecord, error) {
	all, err := p.Actions(ctx)
	if err != nil {
		return nil, err
	}
	var out []model.ActionRecord
	for _, a := range all {
		if a.Namespace == owner && a.RuntimeContext.ExecutionContextID == executionContextID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (p *BaseProvider) Records(ctx context.Context) ([]model.EvidenceRecord, error) {
	return corpus.ListAs[model.EvidenceRecord](ctx, &p.AbstractDocumentProvider, model.KindEvidenceRecord)
}

func (p *BaseProvider) Exceptions(ctx context.Context) ([]model.ExceptionRecord, error) {
	return corpus.ListAs[model.ExceptionRecord](ctx, &p.AbstractDocumentProvider, model.KindExceptionRecord)
}

func (p *BaseProvider) Escalations(ctx context.Context) ([]model.Escalation, error) {
	return corpus.ListAs[model.Escalation](ctx, &p.AbstractDocumentProvider, model.KindEscalation)
}
