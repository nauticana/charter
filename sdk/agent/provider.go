// Package agent defines the contracts a runtime honours for a governed agent: stable identity, versioned definition,
// runtime instance, and the execution context every action is attributed to.
package agent

import (
	"context"

	"github.com/nauticana/charter/sdk/model"
)

type Provider interface {
	Identity(ctx context.Context, ownerNamespace string, ref model.Ref) (model.AgentIdentity, error)
	Definition(ctx context.Context, ownerNamespace string, ref model.Ref) (model.AgentDefinition, error)
	Runtime(ctx context.Context, ownerNamespace string, ref model.Ref) (model.AgentRuntime, error)
	Definitions(ctx context.Context) ([]model.AgentDefinition, error)
	Runtimes(ctx context.Context) ([]model.AgentRuntime, error)
}

// AssignmentSource is the one organization contract admission depends on.
type AssignmentSource interface {
	Assignment(ctx context.Context, ownerNamespace string, ref model.Ref) (model.Assignment, error)
}
