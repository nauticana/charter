package binding

import (
	"context"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// BaseProvider serves binding documents from any corpus.Source.
type BaseProvider struct {
	corpus.AbstractDocumentProvider
}

var _ Provider = (*BaseProvider)(nil)

func NewBaseProvider(s corpus.Source) *BaseProvider {
	return &BaseProvider{corpus.AbstractDocumentProvider{Source: s}}
}

func (p *BaseProvider) System(ctx context.Context, owner string, ref model.Ref) (model.EnterpriseSystem, error) {
	return corpus.ResolveAs[model.EnterpriseSystem](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindEnterpriseSystem)
}

func (p *BaseProvider) Profile(ctx context.Context, owner string, ref model.Ref) (model.SystemProfile, error) {
	return corpus.ResolveAs[model.SystemProfile](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindSystemProfile)
}

func (p *BaseProvider) CapabilityBinding(ctx context.Context, owner string, ref model.Ref) (model.CapabilityBinding, error) {
	return corpus.ResolveAs[model.CapabilityBinding](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindCapabilityBinding)
}

func (p *BaseProvider) DataBinding(ctx context.Context, owner string, ref model.Ref) (model.DataBinding, error) {
	return corpus.ResolveAs[model.DataBinding](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindDataBinding)
}

func (p *BaseProvider) AuthorityBinding(ctx context.Context, owner string, ref model.Ref) (model.AuthorityBinding, error) {
	return corpus.ResolveAs[model.AuthorityBinding](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindAuthorityBinding)
}

func (p *BaseProvider) EventBinding(ctx context.Context, owner string, ref model.Ref) (model.EventBinding, error) {
	return corpus.ResolveAs[model.EventBinding](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindEventBinding)
}

func (p *BaseProvider) Conformance(ctx context.Context, owner string, ref model.Ref) (model.BindingConformance, error) {
	return corpus.ResolveAs[model.BindingConformance](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindBindingConformance)
}

func (p *BaseProvider) CapabilityBindings(ctx context.Context) ([]model.CapabilityBinding, error) {
	return corpus.ListAs[model.CapabilityBinding](ctx, &p.AbstractDocumentProvider, model.KindCapabilityBinding)
}

func (p *BaseProvider) DataBindings(ctx context.Context) ([]model.DataBinding, error) {
	return corpus.ListAs[model.DataBinding](ctx, &p.AbstractDocumentProvider, model.KindDataBinding)
}

func (p *BaseProvider) AuthorityBindings(ctx context.Context) ([]model.AuthorityBinding, error) {
	return corpus.ListAs[model.AuthorityBinding](ctx, &p.AbstractDocumentProvider, model.KindAuthorityBinding)
}

func (p *BaseProvider) EventBindings(ctx context.Context) ([]model.EventBinding, error) {
	return corpus.ListAs[model.EventBinding](ctx, &p.AbstractDocumentProvider, model.KindEventBinding)
}
