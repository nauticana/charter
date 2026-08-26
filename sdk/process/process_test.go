package process

import (
	"context"
	"errors"
	"testing"

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
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"BusinessProcess","id":"P","name":"p","businessOutcome":"o"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"BusinessProcess","id":"P2","name":"p2","businessOutcome":"o"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"Task","id":"T","name":"t","processId":"P","businessOutcome":"o","accountableResponsibilityId":"R","permittedPerformerKinds":["human"],"permittedParticipation":["approves"]}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"ProcessInstance","id":"PI","processId":"P","processDefinitionVersion":"1","state":"running","createdAt":"2026-06-01T00:00:00Z"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"ProcessInstance","id":"PI2","processId":"P2","processDefinitionVersion":"1","state":"running","createdAt":"2026-06-01T00:00:00Z"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"Assignment","id":"A","subject":{"kind":"HumanIdentity","id":"H"},"target":{"kind":"Responsibility","id":"R"},"participation":"approves","validity":{"from":"2026-01-01","to":"2026-03-31"}}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"TaskInstance","id":"TI-AGENT","processInstanceId":"PI","taskId":"T","taskDefinitionVersion":"1","performer":{"kind":"AgentIdentity","id":"G"},"assignmentId":"A","participation":"approves","state":"pending","createdAt":"2026-06-01T00:00:00Z"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"TaskInstance","id":"TI-PART","processInstanceId":"PI","taskId":"T","taskDefinitionVersion":"1","performer":{"kind":"HumanIdentity","id":"H"},"assignmentId":"A","participation":"executes","state":"pending","createdAt":"2026-06-01T00:00:00Z"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"TaskInstance","id":"TI-PROC","processInstanceId":"PI2","taskId":"T","taskDefinitionVersion":"1","performer":{"kind":"HumanIdentity","id":"H"},"assignmentId":"A","participation":"approves","state":"pending","createdAt":"2026-06-01T00:00:00Z"}`,
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
	for id, want := range map[string]error{"TI-AGENT": ErrPerformerKind, "TI-PART": ErrParticipation, "TI-PROC": ErrDefinition, "TI-EXPIRED": ErrAssignment} {
		if _, err := r.TaskContext(context.Background(), "t", model.Ref{ID: id}); !errors.Is(err, want) {
			t.Errorf("%s: got %v, want %v", id, err, want)
		}
	}
}
