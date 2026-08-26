package process

import (
	"context"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// Graph reads hierarchy, sequence, and dependency as the distinct relationships they are declared as (CHR-PROC-003).
type Graph struct {
	Definitions DefinitionProvider
}

// Related returns the objects related to node by one relationship type; inverse returns the subjects that relate to node.
func (g Graph) Related(ctx context.Context, ownerNamespace string, node model.ObjectRef, relationshipType string, inverse bool) ([]model.ObjectRef, error) {
	all, err := g.Definitions.Relationships(ctx)
	if err != nil {
		return nil, err
	}
	want := corpus.ObjectKeyOf(ownerNamespace, node)
	var out []model.ObjectRef
	for _, r := range all {
		if r.RelationshipType != relationshipType {
			continue
		}
		from, to := r.Subject, r.Object
		if inverse {
			from, to = to, from
		}
		if from.Kind == node.Kind && corpus.ObjectKeyOf(r.Namespace, from) == want {
			key := corpus.ObjectKeyOf(r.Namespace, to)
			out = append(out, model.ObjectRef{Kind: to.Kind, ID: key.ID, Namespace: key.Namespace})
		}
	}
	return out, nil
}

func (g Graph) Children(ctx context.Context, owner string, node model.ObjectRef) ([]model.ObjectRef, error) {
	return g.Related(ctx, owner, node, model.RelationshipParentChild, false)
}

func (g Graph) Parents(ctx context.Context, owner string, node model.ObjectRef) ([]model.ObjectRef, error) {
	return g.Related(ctx, owner, node, model.RelationshipParentChild, true)
}

func (g Graph) Predecessors(ctx context.Context, owner string, node model.ObjectRef) ([]model.ObjectRef, error) {
	return g.Related(ctx, owner, node, model.RelationshipPrecedes, true)
}

func (g Graph) Successors(ctx context.Context, owner string, node model.ObjectRef) ([]model.ObjectRef, error) {
	return g.Related(ctx, owner, node, model.RelationshipPrecedes, false)
}

func (g Graph) Dependencies(ctx context.Context, owner string, node model.ObjectRef) ([]model.ObjectRef, error) {
	return g.Related(ctx, owner, node, model.RelationshipDependsOn, false)
}

// TasksOf lists the tasks whose processId names the process; this membership is separate from parent-child relationships.
func (g Graph) TasksOf(ctx context.Context, owner string, process model.Ref) ([]model.Task, error) {
	all, err := g.Definitions.Tasks(ctx)
	if err != nil {
		return nil, err
	}
	want := corpus.KeyOf(owner, process)
	var out []model.Task
	for _, t := range all {
		if corpus.KeyOf(t.Namespace, t.ProcessID) == want {
			out = append(out, t)
		}
	}
	return out, nil
}
