package process

import (
	"errors"
	"fmt"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

var (
	ErrClosed  = errors.New("task instance is closed")
	ErrDue     = errors.New("task instance is due before it was created")
	ErrNotLate = errors.New("task instance is not late")
)

// Assign hands an open instance to a performer under an assignment (CHR-PROC-010).
func Assign(ti model.TaskInstance, performer model.ObjectRef, assignment model.Ref, due *time.Time) (model.TaskInstance, error) {
	switch ti.State {
	case model.TaskCompleted, model.TaskCancelled, model.TaskFailed:
		return model.TaskInstance{}, fmt.Errorf("%w: %s is %s", ErrClosed, ti.ID, ti.State)
	}
	ti.Performer, ti.AssignmentID, ti.DueAt = performer, assignment, due
	if err := validateDue(ti); err != nil {
		return model.TaskInstance{}, err
	}
	return ti, nil
}

func validateDue(ti model.TaskInstance) error {
	if ti.DueAt != nil && ti.DueAt.Before(ti.CreatedAt) {
		return fmt.Errorf("%w: %s", ErrDue, ti.ID)
	}
	return nil
}

// Late reports whether ti was not started by its due time, judged when it started, else ended, else at now (CHR-PROC-010).
func Late(ti model.TaskInstance, now time.Time) bool {
	if ti.DueAt == nil {
		return false
	}
	if ti.StartedAt != nil {
		now = *ti.StartedAt
	} else if ti.EndedAt != nil {
		now = *ti.EndedAt
	}
	return now.After(*ti.DueAt)
}

// LateException escalates a late instance to its task's accountable authority (CHR-PROC-010).
func LateException(tc TaskContext, resp model.Responsibility, id string, now time.Time) (model.ExceptionRecord, error) {
	ti := tc.Instance
	if corpus.KeyOf(resp.Namespace, model.Ref{ID: resp.ID}) != corpus.KeyOf(tc.Task.Namespace, tc.Task.AccountableResponsibilityID) {
		return model.ExceptionRecord{}, fmt.Errorf("%w: %s is not accountable for %s", ErrDefinition, resp.ID, tc.Task.ID)
	}
	if err := validateDue(ti); err != nil {
		return model.ExceptionRecord{}, err
	}
	if !Late(ti, now) {
		return model.ExceptionRecord{}, fmt.Errorf("%w: %s", ErrNotLate, ti.ID)
	}
	target := resp.Accountable
	if target.Namespace == "" {
		target.Namespace = resp.Namespace
	}
	e := model.ExceptionRecord{
		ViolatedExpectation: fmt.Sprintf("task instance %s was not started by %s", ti.ID, ti.DueAt.Format(time.RFC3339)),
		Affected:            model.ObjectRef{Kind: model.KindTaskInstance, ID: ti.ID, Namespace: ti.Namespace},
		Disposition:         "open",
		EscalationTarget:    target,
	}
	e.CharterSpecVersion, e.Namespace, e.ID, e.Kind, e.EnterpriseID = ti.CharterSpecVersion, ti.Namespace, id, model.KindExceptionRecord, ti.EnterpriseID
	return e, nil
}
