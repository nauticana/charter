package organization

import (
	"context"
	"time"

	"github.com/nauticana/charter/sdk/model"
)

type CoverageClass string

const (
	Unassigned     CoverageClass = "unassigned"
	HumanPerformed CoverageClass = "human-performed"
	AgentSupported CoverageClass = "agent-supported"
	AgentExecuted  CoverageClass = "agent-executed"
)

// Coverage classifies how a responsibility is covered at a point in time (CHR-ENT-006).
type Coverage struct {
	Responsibility model.Responsibility
	Class          CoverageClass
	Assignments    []model.Assignment
}

// Coverage classifies the direct, effective assignments of a responsibility; agent execution outranks agent support,
// which outranks human performance. Role membership and position occupancy are never inferred as coverage.
func (q Queries) Coverage(ctx context.Context, ownerNamespace string, responsibility model.Ref, at time.Time) (Coverage, error) {
	resp, err := q.Provider.Responsibility(ctx, ownerNamespace, responsibility)
	if err != nil {
		return Coverage{}, err
	}
	assignments, err := q.AssignmentsTo(ctx, resp.Namespace, model.ObjectRef{Kind: model.KindResponsibility, ID: resp.ID}, at)
	if err != nil {
		return Coverage{}, err
	}
	c := Coverage{Responsibility: resp, Class: Unassigned, Assignments: assignments}
	for _, a := range assignments {
		switch {
		case a.Subject.Kind == model.KindAgentIdentity && a.Participation == model.ParticipationExecutes:
			c.Class = AgentExecuted
		case a.Subject.Kind == model.KindAgentIdentity && c.Class != AgentExecuted:
			c.Class = AgentSupported
		case a.Subject.Kind == model.KindHumanIdentity && c.Class == Unassigned:
			c.Class = HumanPerformed
		}
	}
	return c, nil
}
