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
	"github.com/nauticana/charter/sdk/temporal"
)

var (
	ErrDefinition    = errors.New("instance does not match its definition")
	ErrPerformerKind = errors.New("performer kind is not permitted for the task")
	ErrParticipation = errors.New("participation is not permitted for the task")
	ErrAssignment    = errors.New("assignment does not cover the performer at the instance time")
	ErrBaseline      = errors.New("assignment began before an outcome measure was baselined")
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
	if task.TaskDefinitionVersion != ti.TaskDefinitionVersion {
		return TaskContext{}, fmt.Errorf("%w: task %s is version %s, instance %s runs %s", ErrDefinition, task.ID, task.TaskDefinitionVersion, ti.ID, ti.TaskDefinitionVersion)
	}
	proc, err := r.Process.Process(ctx, pi.Namespace, pi.ProcessID)
	if err != nil {
		return TaskContext{}, err
	}
	if proc.ProcessDefinitionVersion != pi.ProcessDefinitionVersion {
		return TaskContext{}, fmt.Errorf("%w: process %s is version %s, instance %s runs %s", ErrDefinition, proc.ID, proc.ProcessDefinitionVersion, pi.ID, pi.ProcessDefinitionVersion)
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
	if Exceeds(ti.Participation, asg.Participation) {
		return TaskContext{}, fmt.Errorf("%w: %s exceeds %s of %s", ErrParticipation, ti.Participation, asg.Participation, asg.ID)
	}
	if NeedsBaseline(ti) {
		measures, err := r.Process.OutcomeMeasures(ctx)
		if err != nil {
			return TaskContext{}, err
		}
		if ids := Unbaselined(measures, task, asg); len(ids) > 0 {
			return TaskContext{}, fmt.Errorf("%w: %s, measures %v", ErrBaseline, asg.ID, ids)
		}
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

var levels = []string{model.ParticipationObserves, model.ParticipationRecommends, model.ParticipationPrepares, model.ParticipationPerforms, model.ParticipationExecutes}

// Level orders participation from observes to executes (CHR-PROC-008); approves, supports, and occupies are not levels.
func Level(participation string) (int, bool) {
	l := slices.Index(levels, participation)
	return l, l >= 0
}

// Exceeds reports whether participation is a higher level than the one it is bounded by.
func Exceeds(participation, bound string) bool {
	l, ok := Level(participation)
	b, boundOK := Level(bound)
	return ok && boundOK && l > b
}

// NeedsBaseline reports whether an agent performs the instance above recommends, where measures gate it (CHR-PROC-009).
func NeedsBaseline(ti model.TaskInstance) bool {
	return ti.Performer.Kind == model.KindAgentIdentity && Exceeds(ti.Participation, model.ParticipationRecommends)
}

// Unbaselined returns the measures of the task or its process in effect when asg began whose baseline window did not end
// before that day; an unparsable date fails closed.
func Unbaselined(measures []model.OutcomeMeasure, task model.Task, asg model.Assignment) []string {
	var from model.Date
	if asg.Validity != nil {
		from = asg.Validity.From
	}
	start, startErr := time.Parse(time.DateOnly, string(from))
	taskKey, procKey := corpus.KeyOf(task.Namespace, model.Ref{ID: task.ID}), corpus.KeyOf(task.Namespace, task.ProcessID)
	var out []string
	for _, m := range measures {
		key := corpus.ObjectKeyOf(m.Namespace, m.Subject)
		measured := m.Subject.Kind == model.KindTask && key == taskKey || m.Subject.Kind == model.KindBusinessProcess && key == procKey
		active := m.LifecycleState == "" || m.LifecycleState == model.LifecycleActive
		effective := startErr != nil || temporal.EffectiveAt(m.Validity, start)
		if !measured || !active || !effective {
			continue
		}
		baselined := false
		if m.Baseline != nil && startErr == nil {
			end, err := time.Parse(time.DateOnly, string(m.Baseline.Window.To))
			baselined = err == nil && end.Before(start)
		}
		if !baselined {
			out = append(out, m.ID)
		}
	}
	return out
}
