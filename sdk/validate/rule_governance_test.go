package validate

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/authority"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func TestApplicabilityRule(t *testing.T) {
	assignment := func(id string, from model.Date, app *model.Applicability) model.Assignment {
		a := model.Assignment{Envelope: envelope(model.KindAssignment, id), Target: model.ObjectRef{Kind: model.KindPosition, ID: "POS"},
			Participation: model.ParticipationSupports, Applicability: app}
		a.Validity = &model.Validity{From: from, To: "2026-12-31"}
		return a
	}
	lang := func(def bool, values ...string) *model.Applicability {
		return &model.Applicability{Criteria: []model.Criterion{{Dimension: "language", Values: values}}, Default: def}
	}
	rule := ApplicabilityRule{AbstractRule{"CHR-RULE-ENT-006", nil}}
	for name, tc := range map[string]struct {
		a, b model.Assignment
		want int
	}{
		"overlap without default":  {assignment("A", "2026-01-01", lang(false, "en", "de")), assignment("B", "2026-01-01", lang(false, "de")), 1},
		"overlap with one default": {assignment("A", "2026-01-01", lang(true, "en", "de")), assignment("B", "2026-01-01", lang(false, "de")), 0},
		"two defaults":             {assignment("A", "2026-01-01", lang(true, "en")), assignment("B", "2026-01-01", lang(true, "en")), 1},
		"disjoint criteria":        {assignment("A", "2026-01-01", lang(false, "en")), assignment("B", "2026-01-01", lang(false, "de")), 0},
		"unrestricted other":       {assignment("A", "2026-01-01", lang(false, "en")), assignment("B", "2026-01-01", nil), 1},
		"no applicability":         {assignment("A", "2026-01-01", nil), assignment("B", "2026-01-01", nil), 0},
		"successive periods":       {assignment("A", "2025-01-01", lang(false, "en")), assignment("B", "2027-01-01", lang(false, "en")), 0},
	} {
		if got := len(rule.Validate(effectCorpus(t, tc.a, tc.b))); got != tc.want {
			t.Errorf("%s: %d findings, want %d", name, got, tc.want)
		}
	}
}

func TestInstanceParticipationIsBoundedByItsAssignment(t *testing.T) {
	performer := model.ObjectRef{Kind: model.KindAgentIdentity, ID: "AGENT"}
	asg := model.Assignment{Envelope: envelope(model.KindAssignment, "ASGN"), Subject: performer, Target: model.ObjectRef{Kind: model.KindPosition, ID: "POS"},
		Participation: model.ParticipationPrepares}
	asg.Validity = &model.Validity{From: "2026-01-01"}
	rule := InstanceAssignmentRule{AbstractRule{"CHR-RULE-PROC-002", nil}}
	for participation, want := range map[string]int{model.ParticipationObserves: 0, model.ParticipationPrepares: 0, model.ParticipationExecutes: 1} {
		ti := model.TaskInstance{Envelope: envelope(model.KindTaskInstance, "TI"), Performer: performer, AssignmentID: model.Ref{ID: "ASGN"},
			Participation: participation, CreatedAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)}
		if got := len(rule.Validate(effectCorpus(t, asg, ti))); got != want {
			t.Errorf("%s under prepares: %d findings, want %d", participation, got, want)
		}
	}
}

func TestSodApprovalHonorsOnlyANamedValidException(t *testing.T) {
	approver := model.ObjectRef{Kind: model.KindHumanIdentity, ID: "H"}
	sod := model.SodConstraint{Envelope: envelope(model.KindSodConstraint, "SOD"), ConstrainedActions: []model.Ref{{ID: "CAP-PREP"}, {ID: "CAP-FX"}}}
	prepared := model.ActionRecord{Envelope: envelope(model.KindActionRecord, "ACT"), Actor: approver, CapabilityID: model.Ref{ID: "CAP-PREP"},
		AuthorityEvaluations: []model.AuthorityEvaluation{{Result: model.AuthorityAllowed}}}
	approval := model.Approval{Envelope: envelope(model.KindApproval, "APPR"), Approver: approver, ApprovedAction: "CAP-FX", IssuedAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)}
	exception := func(grantor string) model.SodException {
		e := model.SodException{Envelope: envelope(model.KindSodException, "SODEX"), SodConstraintID: model.Ref{ID: "SOD"}, Beneficiary: approver,
			Grantor: model.ObjectRef{Kind: model.KindHumanIdentity, ID: grantor}}
		e.Validity = &model.Validity{From: "2026-05-01", To: "2026-06-30"}
		return e
	}
	rule := SodApprovalRule{AbstractRule{"CHR-RULE-AUTH-003", nil}}
	if f := rule.Validate(effectCorpus(t, sod, prepared, approval, exception("G"))); len(f) != 1 {
		t.Errorf("an exception the approval does not name relieved it: %+v", f)
	}
	approval.SodExceptionID = &model.Ref{ID: "SODEX"}
	if f := rule.Validate(effectCorpus(t, sod, prepared, approval, exception("G"))); len(f) != 0 {
		t.Errorf("named exception: %+v", f)
	}
	if f := rule.Validate(effectCorpus(t, sod, prepared, approval, exception("H"))); len(f) != 1 {
		t.Errorf("self-granted exception relieved the approver: %+v", f)
	}
}

func TestExplanationMatchesItsSchema(t *testing.T) {
	c, err := corpus.NewDirLoader(harbor).Load()
	if err != nil {
		t.Fatal(err)
	}
	d, _ := c.Get("harbor.example", "AUTH-OEC-STOCK-RESERVATION-2026")
	g, err := corpus.Decode[model.AuthorityGrant](d)
	if err != nil {
		t.Fatal(err)
	}
	ev := &authority.AbstractEvaluator{Source: &authority.BaseGrantSource{Items: []model.AuthorityGrant{g}}}
	req := authority.Request{Namespace: g.Namespace, EnterpriseID: *g.EnterpriseID, Actor: g.Actor, CapabilityID: g.CapabilityID, ResourceScope: g.ResourceScope,
		OrganizationalContext: g.OrganizationalContext, At: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)}
	denied := ev.Explain(context.Background(), req)
	req.CapabilityID = model.Ref{ID: "CAP-OTHER"}
	for _, x := range []authority.Explanation{denied, denied.Redact(func(model.ObjectRef) bool { return false }), ev.Explain(context.Background(), req)} {
		b, err := json.Marshal(x)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateFormat("authority-explanation", b); err != nil {
			t.Errorf("%s: %v", b, err)
		}
	}
}

func TestBaselineWindowIsOrdered(t *testing.T) {
	m := model.OutcomeMeasure{Envelope: envelope(model.KindOutcomeMeasure, "M"), Subject: model.ObjectRef{Kind: model.KindTask, ID: "T"},
		Baseline: &model.Baseline{Window: model.Validity{From: "2026-03-31", To: "2026-01-01"}}}
	if f := (BaselineRule{AbstractRule{"CHR-RULE-PROC-004", nil}}).Validate(effectCorpus(t, m)); len(f) != 1 || f[0].DocumentID != "M" {
		t.Errorf("reversed baseline window: %+v", f)
	}
}

func TestLateTaskIsEscalatedToTheAccountableAuthority(t *testing.T) {
	at := func(h int) *time.Time { v := time.Date(2026, 6, 1, h, 0, 0, 0, time.UTC); return &v }
	accountable := model.ObjectRef{Kind: model.KindPosition, ID: "POS"}
	resp := model.Responsibility{Envelope: envelope(model.KindResponsibility, "RESP"), Accountable: accountable}
	task := model.Task{Envelope: envelope(model.KindTask, "T"), AccountableResponsibilityID: model.Ref{ID: "RESP"}}
	instance := func(due, started, ended *time.Time) model.TaskInstance {
		return model.TaskInstance{Envelope: envelope(model.KindTaskInstance, "TI"), TaskID: model.Ref{ID: "T"}, CreatedAt: *at(8), DueAt: due, StartedAt: started, EndedAt: ended}
	}
	exception := func(target model.ObjectRef) model.ExceptionRecord {
		return model.ExceptionRecord{Envelope: envelope(model.KindExceptionRecord, "EXC"), Affected: model.ObjectRef{Kind: model.KindTaskInstance, ID: "TI"}, EscalationTarget: target}
	}
	rule := LateTaskRule{AbstractRule{"CHR-RULE-PROC-005", nil}}
	for name, tc := range map[string]struct {
		docs []any
		want int
	}{
		"no due time":               {[]any{instance(nil, at(12), nil)}, 0},
		"due before creation":       {[]any{instance(at(7), nil, nil)}, 1},
		"started on time":           {[]any{resp, task, instance(at(10), at(10), nil)}, 0},
		"not yet started":           {[]any{resp, task, instance(at(10), nil, nil)}, 0},
		"started late":              {[]any{resp, task, instance(at(10), at(11), nil)}, 1},
		"cancelled late, unstarted": {[]any{resp, task, instance(at(10), nil, at(11))}, 1},
		"started late, escalated":   {[]any{resp, task, exception(accountable), instance(at(10), at(11), nil)}, 0},
		"escalated to another":      {[]any{resp, task, exception(model.ObjectRef{Kind: model.KindPosition, ID: "OTHER"}), instance(at(10), at(11), nil)}, 1},
	} {
		if got := len(rule.Validate(effectCorpus(t, tc.docs...))); got != tc.want {
			t.Errorf("%s: %d findings, want %d", name, got, tc.want)
		}
	}
}
