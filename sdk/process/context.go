package process

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/organization"
)

var (
	ErrDefinition    = errors.New("instance does not match its definition")
	ErrPerformerKind = errors.New("performer kind is not permitted for the task")
	ErrParticipation = errors.New("participation is not permitted for the task")
	ErrAssignment    = errors.New("assignment does not cover the performer at the instance time")
)

// TaskContext is the resolved execution context of one task occurrence; definitions are read, never modified (CHR-PROC-005).
type TaskContext struct {
	Instance        model.TaskInstance
	Task            model.Task
	Process         model.BusinessProcess
	ProcessInstance model.ProcessInstance
	Assignment      model.Assignment
	At              time.Time
}

type ContextResolver interface {
	TaskContext(ctx context.Context, ownerNamespace string, taskInstance model.Ref) (TaskContext, error)
}

// BaseContextResolver resolves task contexts from any Provider and assignment source, failing closed on any inconsistency.
type BaseContextResolver struct {
	Process     Provider
	Assignments AssignmentSource
}

var _ ContextResolver = (*BaseContextResolver)(nil)

func (r *BaseContextResolver) TaskContext(ctx context.Context, owner string, ref model.Ref) (TaskContext, error) {
	ti, err := r.Process.TaskInstance(ctx, owner, ref)
	if err != nil {
		return TaskContext{}, err
	}
	pi, err := r.Process.ProcessInstance(ctx, ti.Namespace, ti.ProcessInstanceID)
	if err != nil {
		return TaskContext{}, err
	}
	task, err := r.Process.Task(ctx, ti.Namespace, ti.TaskID)
	if err != nil {
		return TaskContext{}, err
	}
	if corpus.KeyOf(task.Namespace, task.ProcessID) != corpus.KeyOf(pi.Namespace, pi.ProcessID) {
		return TaskContext{}, fmt.Errorf("%w: task %s belongs to %s, instance %s runs %s", ErrDefinition, task.ID, task.ProcessID.ID, pi.ID, pi.ProcessID.ID)
	}
	proc, err := r.Process.Process(ctx, pi.Namespace, pi.ProcessID)
	if err != nil {
		return TaskContext{}, err
	}
	if !slices.Contains(task.PermittedPerformerKinds, PerformerKind(ti.Performer.Kind)) {
		return TaskContext{}, fmt.Errorf("%w: %s performs %s", ErrPerformerKind, ti.Performer.Kind, task.ID)
	}
	if len(task.PermittedParticipation) > 0 && !slices.Contains(task.PermittedParticipation, ti.Participation) {
		return TaskContext{}, fmt.Errorf("%w: %s %s", ErrParticipation, ti.Participation, task.ID)
	}
	asg, err := r.Assignments.Assignment(ctx, ti.Namespace, ti.AssignmentID)
	if err != nil {
		return TaskContext{}, err
	}
	at := ti.CreatedAt
	if ti.StartedAt != nil {
		at = *ti.StartedAt
	}
	if asg.Subject.Kind != ti.Performer.Kind || corpus.ObjectKeyOf(asg.Namespace, asg.Subject) != corpus.ObjectKeyOf(ti.Namespace, ti.Performer) {
		return TaskContext{}, fmt.Errorf("%w: %s is assigned to %s %s", ErrAssignment, asg.ID, asg.Subject.Kind, asg.Subject.ID)
	}
	if !organization.Current(asg, at) {
		return TaskContext{}, fmt.Errorf("%w: %s is not effective at %s", ErrAssignment, asg.ID, at.Format(time.RFC3339))
	}
	return TaskContext{Instance: ti, Task: task, Process: proc, ProcessInstance: pi, Assignment: asg, At: at}, nil
}

// PerformerKind maps an identity kind to the performer kind a task permits.
func PerformerKind(k model.Kind) string {
	switch k {
	case model.KindHumanIdentity:
		return model.PerformerHuman
	case model.KindAgentIdentity:
		return model.PerformerAgent
	}
	return ""
}
