// Package binding defines provider-neutral contracts for enterprise systems, system profiles, capability, data,
// authority, and event bindings, declared feature support, and binding conformance. Vendor realizations live downstream.
package binding

import (
	"context"

	"github.com/nauticana/charter/sdk/model"
)

type Provider interface {
	System(ctx context.Context, ownerNamespace string, ref model.Ref) (model.EnterpriseSystem, error)
	Profile(ctx context.Context, ownerNamespace string, ref model.Ref) (model.SystemProfile, error)
	CapabilityBinding(ctx context.Context, ownerNamespace string, ref model.Ref) (model.CapabilityBinding, error)
	DataBinding(ctx context.Context, ownerNamespace string, ref model.Ref) (model.DataBinding, error)
	AuthorityBinding(ctx context.Context, ownerNamespace string, ref model.Ref) (model.AuthorityBinding, error)
	EventBinding(ctx context.Context, ownerNamespace string, ref model.Ref) (model.EventBinding, error)
	Conformance(ctx context.Context, ownerNamespace string, ref model.Ref) (model.BindingConformance, error)
	CapabilityBindings(ctx context.Context) ([]model.CapabilityBinding, error)
	DataBindings(ctx context.Context) ([]model.DataBinding, error)
	AuthorityBindings(ctx context.Context) ([]model.AuthorityBinding, error)
	EventBindings(ctx context.Context) ([]model.EventBinding, error)
}
