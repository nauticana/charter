package validate

import (
	"fmt"
	"slices"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/organization"
	"github.com/nauticana/charter/sdk/process"
	"github.com/nauticana/charter/sdk/temporal"
)

type PerformerRule struct{ AbstractRule }
type InstanceAssignmentRule struct{ AbstractRule }
type InstanceDefinitionRule struct{ AbstractRule }
type BaselineRule struct{ AbstractRule }

var (
	_ Rule = PerformerRule{}
	_ Rule = InstanceAssignmentRule{}
	_ Rule = InstanceDefinitionRule{}
	_ Rule = BaselineRule{}
)

func (r PerformerRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, ti := range allOf[model.TaskInstance](c, model.KindTaskInstance) {
		task, ok := resolveAs[model.Task](c, ti.Namespace, ti.TaskID, model.KindTask)
		if !ok {
			continue
		}
		if kind := process.PerformerKind(ti.Performer.Kind); !slices.Contains(task.PermittedPerformerKinds, kind) {
			out = append(out, r.finding(ti.Namespace, ti.ID, fmt.Sprintf("%s %s performs %s, which permits %v", ti.Performer.Kind, ti.Performer.ID, task.ID, task.PermittedPerformerKinds)))
		}
		if len(task.PermittedParticipation) > 0 && !slices.Contains(task.PermittedParticipation, ti.Participation) {
			out = append(out, r.finding(ti.Namespace, ti.ID, fmt.Sprintf("participation %s is not permitted by %s (%v)", ti.Participation, task.ID, task.PermittedParticipation)))
		}
	}
	return out
}

func (r InstanceAssignmentRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, ti := range allOf[model.TaskInstance](c, model.KindTaskInstance) {
		asg, ok := resolveAs[model.Assignment](c, ti.Namespace, ti.AssignmentID, model.KindAssignment)
		if !ok {
			continue
		}
		if asg.Subject.Kind != ti.Performer.Kind || corpus.ObjectKeyOf(asg.Namespace, asg.Subject) != corpus.ObjectKeyOf(ti.Namespace, ti.Performer) {
			out = append(out, r.finding(ti.Namespace, ti.ID, fmt.Sprintf("assignment %s belongs to %s %s, not performer %s %s", asg.ID, asg.Subject.Kind, asg.Subject.ID, ti.Performer.Kind, ti.Performer.ID)))
		}
		at := ti.CreatedAt
		if ti.StartedAt != nil {
			at = *ti.StartedAt
		}
		if !organization.Current(asg, at) {
			out = append(out, r.finding(ti.Namespace, ti.ID, fmt.Sprintf("assignment %s is not effective at %s", asg.ID, at.Format("2006-01-02"))))
		}
		if process.Exceeds(ti.Participation, asg.Participation) {
			out = append(out, r.finding(ti.Namespace, ti.ID, fmt.Sprintf("participation %s exceeds %s of assignment %s", ti.Participation, asg.Participation, asg.ID)))
		}
	}
	return out
}

func (r InstanceDefinitionRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, ti := range allOf[model.TaskInstance](c, model.KindTaskInstance) {
		task, taskOK := resolveAs[model.Task](c, ti.Namespace, ti.TaskID, model.KindTask)
		pi, piOK := resolveAs[model.ProcessInstance](c, ti.Namespace, ti.ProcessInstanceID, model.KindProcessInstance)
		if !taskOK || !piOK {
			continue
		}
		if corpus.KeyOf(task.Namespace, task.ProcessID) != corpus.KeyOf(pi.Namespace, pi.ProcessID) {
			out = append(out, r.finding(ti.Namespace, ti.ID, fmt.Sprintf("task %s belongs to %s, but process instance %s runs %s", task.ID, task.ProcessID.ID, pi.ID, pi.ProcessID.ID)))
		}
	}
	return out
}

func (r BaselineRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	measures := allOf[model.OutcomeMeasure](c, model.KindOutcomeMeasure)
	for _, m := range measures {
		if m.Baseline != nil && !temporal.NewPeriod(m.Baseline.Window).Ordered() {
			out = append(out, r.finding(m.Namespace, m.ID, "baseline window ends before it starts"))
		}
	}
	for _, ti := range allOf[model.TaskInstance](c, model.KindTaskInstance) {
		if !process.NeedsBaseline(ti) {
			continue
		}
		task, taskOK := resolveAs[model.Task](c, ti.Namespace, ti.TaskID, model.KindTask)
		asg, asgOK := resolveAs[model.Assignment](c, ti.Namespace, ti.AssignmentID, model.KindAssignment)
		if !taskOK || !asgOK {
			continue
		}
		if ids := process.Unbaselined(measures, task, asg); len(ids) > 0 {
			out = append(out, r.finding(ti.Namespace, ti.ID, fmt.Sprintf("agent %s under assignment %s, which began before measures %v were baselined", ti.Participation, asg.ID, ids)))
		}
	}
	return out
}
