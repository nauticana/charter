// Package organization resolves enterprise-structure documents and derives assignment, unit-tree, and coverage views.
// It supplies read-only contracts; organization management stays with the application that owns the documents.
package organization

import (
	"context"

	"github.com/nauticana/charter/sdk/model"
)

type Provider interface {
	Enterprise(ctx context.Context, ownerNamespace string, ref model.Ref) (model.Enterprise, error)
	Unit(ctx context.Context, ownerNamespace string, ref model.Ref) (model.OrganizationUnit, error)
	PositionType(ctx context.Context, ownerNamespace string, ref model.Ref) (model.PositionType, error)
	Position(ctx context.Context, ownerNamespace string, ref model.Ref) (model.Position, error)
	Role(ctx context.Context, ownerNamespace string, ref model.Ref) (model.Role, error)
	Responsibility(ctx context.Context, ownerNamespace string, ref model.Ref) (model.Responsibility, error)
	Relationship(ctx context.Context, ownerNamespace string, ref model.Ref) (model.OrganizationRelationship, error)
	Assignment(ctx context.Context, ownerNamespace string, ref model.Ref) (model.Assignment, error)
	Units(ctx context.Context) ([]model.OrganizationUnit, error)
	Assignments(ctx context.Context) ([]model.Assignment, error)
	Relationships(ctx context.Context) ([]model.OrganizationRelationship, error)
}
