package organization

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func queries(t *testing.T, dir string) Queries {
	c, err := corpus.NewDirLoader(dir).Load()
	if err != nil {
		t.Fatal(err)
	}
	return Queries{Provider: NewBaseProvider(c)}
}

func TestHarborQueries(t *testing.T) {
	q := queries(t, "../../examples/harbor-manufacturing/instances")
	ctx := context.Background()
	ns := "harbor.example"
	at := time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)

	path, err := q.UnitPath(ctx, ns, model.Ref{ID: "OU-CREDIT-CONTROL"})
	if err != nil || len(path) != 3 || path[0].ID != "OU-HARBOR" || path[1].ID != "OU-FINANCE" || path[2].ID != "OU-CREDIT-CONTROL" {
		t.Errorf("unit path: %v %v", path, err)
	}
	for _, tc := range []struct {
		position string
		at       time.Time
		want     string
	}{
		{"POS-CREDIT-MANAGER", at, "HUMAN-ALEX-RIVERA"},
		{"PP-PLN-01", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), "HUMAN-PRIYA-SHAH"},
		{"PP-PLN-01", at, "HUMAN-JORDAN-KIM"},
	} {
		occupants, err := q.Occupants(ctx, ns, model.Ref{ID: tc.position}, tc.at)
		if err != nil || len(occupants) != 1 || occupants[0].Subject.ID != tc.want {
			t.Errorf("occupants of %s at %s: %v %v", tc.position, tc.at.Format("2006-01-02"), occupants, err)
		}
	}
	agent := model.ObjectRef{Kind: model.KindAgentIdentity, ID: "AGENT-PLANNING-DATA-QUALITY"}
	if of, err := q.AssignmentsOf(ctx, ns, agent, at); err != nil || len(of) != 2 {
		t.Errorf("agent assignments: %v %v", of, err)
	}
	if of, err := q.AssignmentsOf(ctx, ns, agent, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)); err != nil || len(of) != 0 {
		t.Errorf("agent assignments before validity: %v %v", of, err)
	}
	if c, err := q.Coverage(ctx, ns, model.Ref{ID: "RESP-APPROVE-CREDIT-EXCEPTION"}, at); err != nil || c.Class != HumanPerformed {
		t.Errorf("credit approval coverage: %+v %v", c.Class, err)
	}
	if c, err := q.Coverage(ctx, ns, model.Ref{ID: "RESP-COORDINATE-ORDER-EXCEPTION"}, at); err != nil || c.Class != Unassigned {
		t.Errorf("work-context support must not count as responsibility coverage: %+v %v", c.Class, err)
	}
}

func TestCoverageClassPrecedence(t *testing.T) {
	c := corpus.New()
	for _, raw := range []string{
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"Responsibility","id":"R","name":"r","expectedOutcome":"o","accountable":{"kind":"Position","id":"P"}}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"Assignment","id":"A1","subject":{"kind":"HumanIdentity","id":"H"},"target":{"kind":"Responsibility","id":"R"},"participation":"performs","validity":{"from":"2026-01-01"}}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"Assignment","id":"A2","subject":{"kind":"AgentIdentity","id":"G"},"target":{"kind":"Responsibility","id":"R"},"participation":"executes","validity":{"from":"2026-01-01"},"lifecycleState":"retired"}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"Assignment","id":"A3","subject":{"kind":"AgentIdentity","id":"G"},"target":{"kind":"Responsibility","id":"R"},"participation":"prepares","validity":{"from":"2026-01-01"}}`,
	} {
		d, err := (corpus.Parser{}).Parse([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if err := c.Add(d); err != nil {
			t.Fatal(err)
		}
	}
	cov, err := (Queries{Provider: NewBaseProvider(c)}).Coverage(context.Background(), "t", model.Ref{ID: "R"}, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || cov.Class != AgentSupported || len(cov.Assignments) != 2 {
		t.Fatalf("got %s over %d assignments, want agent-supported over 2 (retired execution excluded): %v", cov.Class, len(cov.Assignments), err)
	}
}

func TestUnitPathDetectsCycle(t *testing.T) {
	q := queries(t, "../../conformance/fixtures/invalid/cyclic-unit-tree")
	if _, err := q.UnitPath(context.Background(), "fixture.example", model.Ref{ID: "OU-A"}); !errors.Is(err, ErrCycle) {
		t.Fatalf("cycle not detected: %v", err)
	}
}
