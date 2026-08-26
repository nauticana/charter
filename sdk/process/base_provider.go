package process

import (
	"context"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// BaseProvider serves process documents from any corpus.Source.
type BaseProvider struct {
	corpus.AbstractDocumentProvider
}

var _ Provider = (*BaseProvider)(nil)

func NewBaseProvider(s corpus.Source) *BaseProvider {
	return &BaseProvider{corpus.AbstractDocumentProvider{Source: s}}
}

func (p *BaseProvider) ValueStream(ctx context.Context, owner string, ref model.Ref) (model.ValueStream, error) {
	return corpus.ResolveAs[model.ValueStream](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindValueStream)
}

func (p *BaseProvider) Process(ctx context.Context, owner string, ref model.Ref) (model.BusinessProcess, error) {
	return corpus.ResolveAs[model.BusinessProcess](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindBusinessProcess)
}

func (p *BaseProvider) Task(ctx context.Context, owner string, ref model.Ref) (model.Task, error) {
	return corpus.ResolveAs[model.Task](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindTask)
}

func (p *BaseProvider) Relationship(ctx context.Context, owner string, ref model.Ref) (model.ProcessRelationship, error) {
	return corpus.ResolveAs[model.ProcessRelationship](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindProcessRelationship)
}

func (p *BaseProvider) Tasks(ctx context.Context) ([]model.Task, error) {
	return corpus.ListAs[model.Task](ctx, &p.AbstractDocumentProvider, model.KindTask)
}

func (p *BaseProvider) Relationships(ctx context.Context) ([]model.ProcessRelationship, error) {
	return corpus.ListAs[model.ProcessRelationship](ctx, &p.AbstractDocumentProvider, model.KindProcessRelationship)
}

func (p *BaseProvider) ProcessInstance(ctx context.Context, owner string, ref model.Ref) (model.ProcessInstance, error) {
	return corpus.ResolveAs[model.ProcessInstance](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindProcessInstance)
}

func (p *BaseProvider) TaskInstance(ctx context.Context, owner string, ref model.Ref) (model.TaskInstance, error) {
	return corpus.ResolveAs[model.TaskInstance](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindTaskInstance)
}

func (p *BaseProvider) TaskInstances(ctx context.Context) ([]model.TaskInstance, error) {
	return corpus.ListAs[model.TaskInstance](ctx, &p.AbstractDocumentProvider, model.KindTaskInstance)
}
