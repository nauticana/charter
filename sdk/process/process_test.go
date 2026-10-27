package process

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/organization"
)

func TestHarborGraphAndContext(t *testing.T) {
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ns := "harbor.example"
	p := NewBaseProvider(c)
	g := Graph{Definitions: p}
	if pre, err := g.Predecessors(ctx, ns, model.ObjectRef{Kind: model.KindTask, ID: "TASK-OE-EXECUTE"}); err != nil || len(pre) != 1 || pre[0].ID != "TASK-OE-APPROVE" || pre[0].Namespace != ns {
		t.Errorf("predecessors: %v %v", pre, err)
	}
	if next, err := g.Successors(ctx, ns, model.ObjectRef{Kind: model.KindTask, ID: "TASK-OE-APPROVE"}); err != nil || len(next) != 1 || next[0].ID != "TASK-OE-EXECUTE" {
		t.Errorf("successors: %v %v", next, err)
	}
	if children, err := g.Children(ctx, ns, model.ObjectRef{Kind: model.KindBusinessProcess, ID: "PROC-RESOLVE-ORDER-EXCEPTION"}); err != nil || len(children) != 1 || children[0].ID != "TASK-OE-DETECT" {
		t.Errorf("children: %v %v", children, err)
	}
	if tasks, err := g.TasksOf(ctx, ns, model.Ref{ID: "PROC-RESOLVE-ORDER-EXCEPTION"}); err != nil || len(tasks) != 6 {
		t.Errorf("tasks of process: %d %v", len(tasks), err)
	}
	r := &BaseContextResolver{Process: p, Assignments: organization.NewBaseProvider(c)}
	for _, id := range []string{"TASKINST-OE-0042-EXECUTE", "TASKINST-OE-0042-APPROVE", "TASKINST-OE-0042-DETECT"} {
		tc, err := r.TaskContext(ctx, ns, model.Ref{ID: id})
		if err != nil || tc.Process.ID != "PROC-RESOLVE-ORDER-EXCEPTION" || tc.Assignment.ID == "" {
			t.Errorf("%s: %+v %v", id, tc.Assignment.ID, err)
		}
	}
}

func TestContextFailsClosed(t *testing.T) {
	docs := []string{
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"BusinessProcess","id":"P","name":"p","businessOutcome":"o","processDefinitionVersion":"1"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"BusinessProcess","id":"P2","name":"p2","businessOutcome":"o","processDefinitionVersion":"1"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"Task","id":"T","name":"t","processId":"P","taskDefinitionVersion":"1","businessOutcome":"o","accountableResponsibilityId":"R","permittedPerformerKinds":["human"],"permittedParticipation":["approves"]}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"ProcessInstance","id":"PI","processId":"P","processDefinitionVersion":"1","state":"running","createdAt":"2026-06-01T00:00:00Z"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"ProcessInstance","id":"PI2","processId":"P2","processDefinitionVersion":"1","state":"running","createdAt":"2026-06-01T00:00:00Z"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"Assignment","id":"A","subject":{"kind":"HumanIdentity","id":"H"},"target":{"kind":"Responsibility","id":"R"},"participation":"approves","validity":{"from":"2026-01-01","to":"2026-03-31"}}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"TaskInstance","id":"TI-AGENT","processInstanceId":"PI","taskId":"T","taskDefinitionVersion":"1","performer":{"kind":"AgentIdentity","id":"G"},"assignmentId":"A","participation":"approves","state":"pending","createdAt":"2026-06-01T00:00:00Z"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"TaskInstance","id":"TI-PART","processInstanceId":"PI","taskId":"T","taskDefinitionVersion":"1","performer":{"kind":"HumanIdentity","id":"H"},"assignmentId":"A","participation":"executes","state":"pending","createdAt":"2026-06-01T00:00:00Z"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"TaskInstance","id":"TI-PROC","processInstanceId":"PI2","taskId":"T","taskDefinitionVersion":"1","performer":{"kind":"HumanIdentity","id":"H"},"assignmentId":"A","participation":"approves","state":"pending","createdAt":"2026-06-01T00:00:00Z"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"TaskInstance","id":"TI-VERSION","processInstanceId":"PI","taskId":"T","taskDefinitionVersion":"2","performer":{"kind":"HumanIdentity","id":"H"},"assignmentId":"A","participation":"approves","state":"pending","createdAt":"2026-06-01T00:00:00Z"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"Task","id":"T-EXEC","name":"t","processId":"P","taskDefinitionVersion":"1","businessOutcome":"o","accountableResponsibilityId":"R","permittedPerformerKinds":["human"],"permittedParticipation":["executes"]}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"Assignment","id":"A-PREP","subject":{"kind":"HumanIdentity","id":"H"},"target":{"kind":"Responsibility","id":"R"},"participation":"prepares","validity":{"from":"2026-01-01"}}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"TaskInstance","id":"TI-LEVEL","processInstanceId":"PI","taskId":"T-EXEC","taskDefinitionVersion":"1","performer":{"kind":"HumanIdentity","id":"H"},"assignmentId":"A-PREP","participation":"executes","state":"pending","createdAt":"2026-06-01T00:00:00Z"}`,
		`{"charterSpecVersion":"1.4.0","namespace":"t","kind":"Assignment","id":"A-AGENT","subject":{"kind":"AgentIdentity","id":"G"},"target":{"kind":"Responsibility","id":"R"},"participation":"executes","validity":{"from":"2026-04-01"}}`,
		`{"charterSpecVersion":"1.4.0","namespace":"t","kind":"Task","id":"T-AGENT","name":"t","processId":"P","taskDefinitionVersion":"1","businessOutcome":"o","accountableResponsibilityId":"R","permittedPerformerKinds":["agent"]}`,
		`{"charterSpecVersion":"1.4.0","namespace":"t","kind":"OutcomeMeasure","id":"M","name":"m","subject":{"kind":"BusinessProcess","id":"P"},"improvement":"decrease","unit":"hour","baseline":{"value":26,"window":{"from":"2026-01-01","to":"2026-04-01"},"evidenceRecordIds":["E"]}}`,
		`{"charterSpecVersion":"1.4.0","namespace":"t","kind":"TaskInstance","id":"TI-BASELINE","processInstanceId":"PI","taskId":"T-AGENT","taskDefinitionVersion":"1","performer":{"kind":"AgentIdentity","id":"G"},"assignmentId":"A-AGENT","participation":"executes","state":"pending","createdAt":"2026-06-01T00:00:00Z"}`,
		`{"charterSpecVersion":"1.5.0","namespace":"t","kind":"TaskInstance","id":"TI-DUE","processInstanceId":"PI","taskId":"T","taskDefinitionVersion":"1","performer":{"kind":"HumanIdentity","id":"H"},"assignmentId":"A","participation":"approves","state":"pending","createdAt":"2026-06-01T00:00:00Z","dueAt":"2026-05-31T00:00:00Z"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"TaskInstance","id":"TI-EXPIRED","processInstanceId":"PI","taskId":"T","taskDefinitionVersion":"1","performer":{"kind":"HumanIdentity","id":"H"},"assignmentId":"A","participation":"approves","state":"pending","createdAt":"2026-06-01T00:00:00Z"}`,
	}
	c := corpus.New()
	for _, raw := range docs {
		d, err := (corpus.Parser{}).Parse([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if err := c.Add(d); err != nil {
			t.Fatal(err)
		}
	}
	r := &BaseContextResolver{Process: NewBaseProvider(c), Assignments: organization.NewBaseProvider(c)}
	for id, want := range map[string]error{"TI-AGENT": ErrPerformerKind, "TI-PART": ErrParticipation, "TI-PROC": ErrDefinition, "TI-VERSION": ErrDefinition, "TI-EXPIRED": ErrAssignment, "TI-LEVEL": ErrParticipation, "TI-BASELINE": ErrBaseline, "TI-DUE": ErrDue} {
		if _, err := r.TaskContext(context.Background(), "t", model.Ref{ID: id}); !errors.Is(err, want) {
			t.Errorf("%s: got %v, want %v", id, err, want)
		}
	}
}

func TestParticipationLevels(t *testing.T) {
	if !Exceeds(model.ParticipationExecutes, model.ParticipationPerforms) || Exceeds(model.ParticipationObserves, model.ParticipationRecommends) {
		t.Error("levels out of order")
	}
	if Exceeds(model.ParticipationExecutes, model.ParticipationSupports) || Exceeds(model.ParticipationApproves, model.ParticipationObserves) {
		t.Error("supports and approves are not levels")
	}
}

func TestUnbaselined(t *testing.T) {
	task := model.Task{Namespace: "t", ID: "T", ProcessID: model.Ref{ID: "P"}}
	measure := func(kind model.Kind, id string, to model.Date, edit func(*model.OutcomeMeasure)) model.OutcomeMeasure {
		m := model.OutcomeMeasure{Namespace: "t", ID: "M", Subject: model.ObjectRef{Kind: kind, ID: id}}
		if to != "" {
			m.Baseline = &model.Baseline{Window: model.Validity{From: "2026-01-01", To: to}}
		}
		if edit != nil {
			edit(&m)
		}
		return m
	}
	for name, tc := range map[string]struct {
		m     model.OutcomeMeasure
		start model.Date
		want  bool
	}{
		"task, no baseline":                {measure(model.KindTask, "T", "", nil), "2026-04-01", true},
		"process, baselined":               {measure(model.KindBusinessProcess, "P", "2026-03-31", nil), "2026-04-01", false},
		"window ends on start":             {measure(model.KindTask, "T", "2026-04-01", nil), "2026-04-01", true},
		"window end unparsable":            {measure(model.KindTask, "T", "March", nil), "2026-04-01", true},
		"start unparsable":                 {measure(model.KindTask, "T", "2026-03-31", nil), "April", true},
		"other task":                       {measure(model.KindTask, "T2", "", nil), "2026-04-01", false},
		"task id named as process":         {measure(model.KindBusinessProcess, "T", "", nil), "2026-04-01", false},
		"value stream":                     {measure(model.KindValueStream, "P", "", nil), "2026-04-01", false},
		"other namespace":                  {measure(model.KindTask, "T", "", func(m *model.OutcomeMeasure) { m.Subject.Namespace = "u" }), "2026-04-01", false},
		"retired":                          {measure(model.KindTask, "T", "", func(m *model.OutcomeMeasure) { m.LifecycleState = model.LifecycleRetired }), "2026-04-01", false},
		"declared after the start":         {measure(model.KindTask, "T", "", func(m *model.OutcomeMeasure) { m.Validity = &model.Validity{From: "2026-05-01"} }), "2026-04-01", false},
		"in effect on the start":           {measure(model.KindTask, "T", "", func(m *model.OutcomeMeasure) { m.Validity = &model.Validity{From: "2026-04-01"} }), "2026-04-01", true},
		"start unparsable, declared later": {measure(model.KindTask, "T", "2026-03-31", func(m *model.OutcomeMeasure) { m.Validity = &model.Validity{From: "2026-05-01"} }), "April", true},
	} {
		asg := model.Assignment{Validity: &model.Validity{From: tc.start}}
		if got := len(Unbaselined([]model.OutcomeMeasure{tc.m}, task, asg)) == 1; got != tc.want {
			t.Errorf("%s: unbaselined %t, want %t", name, got, tc.want)
		}
	}
	if len(Unbaselined([]model.OutcomeMeasure{measure(model.KindTask, "T", "2026-03-31", nil)}, task, model.Assignment{})) != 1 {
		t.Error("an assignment without validity fails closed")
	}
	agent := model.TaskInstance{Performer: model.ObjectRef{Kind: model.KindAgentIdentity}, Participation: model.ParticipationPrepares}
	human, advice := agent, agent
	human.Performer.Kind = model.KindHumanIdentity
	advice.Participation = model.ParticipationRecommends
	if !NeedsBaseline(agent) || NeedsBaseline(human) || NeedsBaseline(advice) {
		t.Error("only agent work above recommends needs a baseline")
	}
}

func TestAssignAndLate(t *testing.T) {
	at := func(h int) *time.Time { v := time.Date(2026, 6, 1, h, 0, 0, 0, time.UTC); return &v }
	open := model.TaskInstance{Namespace: "t", ID: "TI", State: model.TaskPending, CreatedAt: *at(8)}
	performer := model.ObjectRef{Kind: model.KindHumanIdentity, ID: "H"}
	got, err := Assign(open, performer, model.Ref{ID: "A"}, at(10))
	if err != nil || got.Performer != performer || got.AssignmentID.ID != "A" || !got.DueAt.Equal(*at(10)) || open.DueAt != nil {
		t.Fatalf("assign: %+v %v", got, err)
	}
	if _, err := Assign(open, performer, model.Ref{ID: "A"}, at(7)); !errors.Is(err, ErrDue) {
		t.Errorf("due before creation: %v", err)
	}
	for _, state := range []string{model.TaskCompleted, model.TaskCancelled, model.TaskFailed} {
		closed := open
		closed.State = state
		if _, err := Assign(closed, performer, model.Ref{ID: "A"}, nil); !errors.Is(err, ErrClosed) {
			t.Errorf("%s: %v", state, err)
		}
	}
	if Late(open, *at(23)) || Late(got, *at(10)) || !Late(got, *at(11)) {
		t.Error("late only after the due time")
	}
	started := got
	started.StartedAt = at(9)
	if Late(started, *at(23)) {
		t.Error("started on time is never late")
	}
	ended := got
	ended.EndedAt = at(11)
	if !Late(ended, *at(9)) {
		t.Error("ended unstarted after the due time is late")
	}
}

func TestLateException(t *testing.T) {
	due := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	enterprise := model.Ref{ID: "ENT"}
	tc := TaskContext{
		Instance: model.TaskInstance{Envelope: model.Envelope{CharterSpecVersion: "1.5.0", Namespace: "t", ID: "TI", EnterpriseID: &enterprise}, DueAt: &due},
		Task:     model.Task{Envelope: model.Envelope{Namespace: "t", ID: "T"}, AccountableResponsibilityID: model.Ref{ID: "R"}},
	}
	resp := model.Responsibility{Namespace: "t", ID: "R", Accountable: model.ObjectRef{Kind: model.KindPosition, ID: "POS"}}
	e, err := LateException(tc, resp, "EXC", due.Add(time.Hour))
	if err != nil || e.ID != "EXC" || e.Kind != model.KindExceptionRecord || e.Affected != (model.ObjectRef{Kind: model.KindTaskInstance, ID: "TI", Namespace: "t"}) ||
		e.EscalationTarget != (model.ObjectRef{Kind: model.KindPosition, ID: "POS", Namespace: "t"}) || e.EnterpriseID != &enterprise || e.Disposition == "" {
		t.Fatalf("exception: %+v %v", e, err)
	}
	other := resp
	other.ID = "R2"
	if _, err := LateException(tc, other, "EXC", due.Add(time.Hour)); !errors.Is(err, ErrDefinition) {
		t.Errorf("another responsibility: %v", err)
	}
	if _, err := LateException(tc, resp, "EXC", due); !errors.Is(err, ErrNotLate) {
		t.Errorf("not late: %v", err)
	}
	tc.Instance.DueAt = nil
	if _, err := LateException(tc, resp, "EXC", due.Add(time.Hour)); !errors.Is(err, ErrNotLate) {
		t.Errorf("no due time: %v", err)
	}
}
