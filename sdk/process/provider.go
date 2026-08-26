// Package process resolves value streams, process and task definitions, relationships, and instances while keeping
// definitions and execution context distinct.
package process

import (
	"context"

	"github.com/nauticana/charter/sdk/model"
)

type DefinitionProvider interface {
	ValueStream(ctx context.Context, ownerNamespace string, ref model.Ref) (model.ValueStream, error)
	Process(ctx context.Context, ownerNamespace string, ref model.Ref) (model.BusinessProcess, error)
	Task(ctx context.Context, ownerNamespace string, ref model.Ref) (model.Task, error)
	Relationship(ctx context.Context, ownerNamespace string, ref model.Ref) (model.ProcessRelationship, error)
	Tasks(ctx context.Context) ([]model.Task, error)
	Relationships(ctx context.Context) ([]model.ProcessRelationship, error)
}

type InstanceProvider interface {
	ProcessInstance(ctx context.Context, ownerNamespace string, ref model.Ref) (model.ProcessInstance, error)
	TaskInstance(ctx context.Context, ownerNamespace string, ref model.Ref) (model.TaskInstance, error)
	TaskInstances(ctx context.Context) ([]model.TaskInstance, error)
}

type Provider interface {
	DefinitionProvider
	InstanceProvider
}

// AssignmentSource is the one organization contract task-context resolution depends on.
type AssignmentSource interface {
	Assignment(ctx context.Context, ownerNamespace string, ref model.Ref) (model.Assignment, error)
}
