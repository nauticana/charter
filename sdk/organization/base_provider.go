package organization

import (
	"context"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// BaseProvider serves organization documents from any corpus.Source.
type BaseProvider struct {
	corpus.AbstractDocumentProvider
}

var _ Provider = (*BaseProvider)(nil)

func NewBaseProvider(s corpus.Source) *BaseProvider {
	return &BaseProvider{corpus.AbstractDocumentProvider{Source: s}}
}

func (p *BaseProvider) Enterprise(ctx context.Context, owner string, ref model.Ref) (model.Enterprise, error) {
	return corpus.ResolveAs[model.Enterprise](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindEnterprise)
}

func (p *BaseProvider) Unit(ctx context.Context, owner string, ref model.Ref) (model.OrganizationUnit, error) {
	return corpus.ResolveAs[model.OrganizationUnit](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindOrganizationUnit)
}

func (p *BaseProvider) PositionType(ctx context.Context, owner string, ref model.Ref) (model.PositionType, error) {
	return corpus.ResolveAs[model.PositionType](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindPositionType)
}

func (p *BaseProvider) Position(ctx context.Context, owner string, ref model.Ref) (model.Position, error) {
	return corpus.ResolveAs[model.Position](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindPosition)
}

func (p *BaseProvider) Role(ctx context.Context, owner string, ref model.Ref) (model.Role, error) {
	return corpus.ResolveAs[model.Role](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindRole)
}

func (p *BaseProvider) Responsibility(ctx context.Context, owner string, ref model.Ref) (model.Responsibility, error) {
	return corpus.ResolveAs[model.Responsibility](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindResponsibility)
}

func (p *BaseProvider) Relationship(ctx context.Context, owner string, ref model.Ref) (model.OrganizationRelationship, error) {
	return corpus.ResolveAs[model.OrganizationRelationship](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindOrganizationRelationship)
}

func (p *BaseProvider) Assignment(ctx context.Context, owner string, ref model.Ref) (model.Assignment, error) {
	return corpus.ResolveAs[model.Assignment](ctx, &p.AbstractDocumentProvider, owner, ref, model.KindAssignment)
}

func (p *BaseProvider) Units(ctx context.Context) ([]model.OrganizationUnit, error) {
	return corpus.ListAs[model.OrganizationUnit](ctx, &p.AbstractDocumentProvider, model.KindOrganizationUnit)
}

func (p *BaseProvider) Assignments(ctx context.Context) ([]model.Assignment, error) {
	return corpus.ListAs[model.Assignment](ctx, &p.AbstractDocumentProvider, model.KindAssignment)
}

func (p *BaseProvider) Relationships(ctx context.Context) ([]model.OrganizationRelationship, error) {
	return corpus.ListAs[model.OrganizationRelationship](ctx, &p.AbstractDocumentProvider, model.KindOrganizationRelationship)
}
