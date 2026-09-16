package agent

import (
	"context"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/organization"
)

func TestBaseAdmission(t *testing.T) {
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	admission := &BaseAdmission{Agents: NewBaseProvider(c), Assignments: organization.NewBaseProvider(c)}
	capability := model.Ref{ID: "CAP-RESERVE-ORDER-STOCK"}
	base := ExecutionContext{
		Namespace: "harbor.example", Identity: model.Ref{ID: "AGENT-ORDER-EXCEPTION-COORDINATOR"}, Runtime: model.Ref{ID: "RT-OEC-PROD-01"},
		ExecutionContextID: "EXEC-OE-0042-05", Definition: model.Ref{ID: "AGENTDEF-ORDER-EXCEPTION-COORDINATOR-1"}, DefinitionVersion: "1",
		Assignment: model.Ref{ID: "ASGN-OEC-ORDER-EXCEPTION-SUPPORT"}, Participation: model.ParticipationSupports, Capability: &capability,
		At: time.Date(2026, 6, 18, 17, 12, 30, 0, time.UTC),
	}
	cases := []struct {
		name        string
		mutate      func(*ExecutionContext)
		want        Result
		requirement string
	}{
		{"harbor context", func(*ExecutionContext) {}, Admitted, ""},
		{"other definition version", func(e *ExecutionContext) { e.DefinitionVersion = "2" }, Refused, "CHR-AGENT-006"},
		{"undeclared capability", func(e *ExecutionContext) { e.Capability = &model.Ref{ID: "CAP-APPROVE-CREDIT-EXCEPTION"} }, Refused, "CHR-AGENT-003"},
		{"assignment expired", func(e *ExecutionContext) { e.At = time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC) }, Refused, "CHR-AGENT-007"},
		{"before identity activation", func(e *ExecutionContext) { e.At = time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC) }, Refused, "CHR-ID-005"},
		{"another subject's assignment", func(e *ExecutionContext) { e.Assignment = model.Ref{ID: "ASGN-ALEX-CREDIT-MANAGER"} }, Refused, "CHR-AGENT-007"},
		{"participation not assigned", func(e *ExecutionContext) { e.Participation = model.ParticipationExecutes }, Refused, "CHR-PROC-006"},
		{"unknown runtime", func(e *ExecutionContext) { e.Runtime = model.Ref{ID: "RT-NOWHERE"} }, Error, "CHR-AGENT-006"},
	}
	for _, tc := range cases {
		ec := base
		tc.mutate(&ec)
		d := admission.Admit(context.Background(), ec)
		if d.Result != tc.want || d.Requirement != tc.requirement {
			t.Errorf("%s: got %s %s (%s), want %s %s", tc.name, d.Result, d.Requirement, d.Reason, tc.want, tc.requirement)
		}
	}
	if rc := base.RuntimeContext(); rc.RuntimeInstanceID.ID != "RT-OEC-PROD-01" || rc.ExecutionContextID != "EXEC-OE-0042-05" || base.Actor().Kind != model.KindAgentIdentity {
		t.Error("execution context does not project runtime context and actor")
	}
}

func TestTriggered(t *testing.T) {
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	def, err := NewBaseProvider(c).Definition(context.Background(), "harbor.example", model.Ref{ID: "AGENTDEF-ORDER-EXCEPTION-COORDINATOR-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !Triggered(def, "order-blocked event mapped to a known order") || Triggered(def, "price change") {
		t.Error("declared triggers not recognised")
	}
}

// A definition that does not declare the version its runtime operates is refused: without this the runtime's
// version string is an unverified claim and an edited definition passes admission unnoticed.
func TestAdmissionRefusesDefinitionVersionDrift(t *testing.T) {
	docs := []string{
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"AgentIdentity","id":"G","name":"g","lifecycleState":"active","lifecycleHistory":[{"state":"active","effectiveAt":"2026-01-01T00:00:00Z"}]}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"AgentDefinition","id":"D","name":"d","enterpriseId":"E","agentIdentityId":"G","definitionVersion":"1","purpose":"p","accountable":{"kind":"Position","id":"POS"},"responsibilityIds":["R"],"triggers":["t"],"requiredInputs":["i"],"expectedOutcomes":["o"],"capabilityIds":[],"policies":[],"confidenceBoundaries":[],"escalationConditions":[]}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"AgentRuntime","id":"RT","name":"rt","lifecycleState":"active","agentIdentityId":"G","agentDefinitionId":"D","definitionVersion":"2","operator":{"kind":"OrganizationUnit","id":"OU"}}`,
		`{"charterSpecVersion":"1.0.0","namespace":"t","kind":"Assignment","id":"A","subject":{"kind":"AgentIdentity","id":"G"},"target":{"kind":"Responsibility","id":"R"},"participation":"supports","validity":{"from":"2026-01-01"}}`,
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
	admission := &BaseAdmission{Agents: NewBaseProvider(c), Assignments: organization.NewBaseProvider(c)}
	ec := ExecutionContext{
		Namespace: "t", Identity: model.Ref{ID: "G"}, Runtime: model.Ref{ID: "RT"}, ExecutionContextID: "X",
		Definition: model.Ref{ID: "D"}, DefinitionVersion: "2", Assignment: model.Ref{ID: "A"},
		Participation: model.ParticipationSupports, At: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	d := admission.Admit(context.Background(), ec)
	if d.Result != Refused || d.Requirement != "CHR-AGENT-006" {
		t.Fatalf("drift = %s %s (%s), want Refused CHR-AGENT-006", d.Result, d.Requirement, d.Reason)
	}
}
