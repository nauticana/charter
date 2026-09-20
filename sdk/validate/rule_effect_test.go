package validate

import (
	"strings"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func effectCorpus(t *testing.T, documents ...any) *corpus.Corpus {
	t.Helper()
	c := corpus.New()
	for _, d := range documents {
		mediationDocument(t, c, d)
	}
	return c
}

func envelope(kind model.Kind, id string) model.Envelope {
	return model.Envelope{Namespace: "test.example", ID: id, Kind: kind}
}

func TestVerifiedEffectRule(t *testing.T) {
	contract := model.CapabilityContract{Envelope: envelope(model.KindCapabilityContract, "CAP"), Outcomes: []string{"done", "rejected"}, BusinessErrors: []string{"refused"},
		Postconditions: []model.Postcondition{{ID: "POST", Statement: "held", VerificationRequired: true, Outcomes: []string{"done"}, ViolationBusinessError: "refused"}}}
	observed := model.EvidenceRecord{Envelope: envelope(model.KindEvidenceRecord, "EVR"), Category: model.CategoryObservedFact}
	action := func(outcome, result string) model.ActionRecord {
		a := model.ActionRecord{Envelope: envelope(model.KindActionRecord, "ACT"), CapabilityID: model.Ref{ID: "CAP"}, Outcome: outcome, Disposition: model.DispositionExecuted}
		if result != "" {
			a.PostconditionEvaluations = []model.PostconditionEvaluation{{PostconditionID: "POST", Result: result, EvidenceRecordIDs: []model.Ref{{ID: "EVR"}}}}
		}
		return a
	}
	rule := VerifiedEffectRule{AbstractRule{"CHR-RULE-CAP-002", nil}}
	for name, tc := range map[string]struct {
		documents []any
		want      string
	}{
		"satisfied":                         {[]any{contract, observed, action("done", model.PostconditionSatisfied)}, ""},
		"outcome outside the postcondition": {[]any{contract, action("rejected", "")}, ""},
		"not evaluated":                     {[]any{contract, action("done", "")}, "POST is not evaluated"},
		"violated":                          {[]any{contract, observed, action("done", model.PostconditionViolated)}, "POST is violated"},
		"acknowledgement as effect evidence": {[]any{contract, model.EvidenceRecord{Envelope: envelope(model.KindEvidenceRecord, "EVR"), Category: model.CategoryExternalResponse},
			action("done", model.PostconditionSatisfied)}, "rather than an observed fact"},
		"undeclared evaluation": {[]any{model.CapabilityContract{Envelope: envelope(model.KindCapabilityContract, "CAP"), Outcomes: []string{"done"}}, observed,
			action("done", model.PostconditionSatisfied)}, "does not declare"},
	} {
		findings := rule.Validate(effectCorpus(t, tc.documents...))
		if (tc.want == "") != (len(findings) == 0) || (tc.want != "" && (len(findings) != 1 || !strings.Contains(findings[0].Message, tc.want))) {
			t.Errorf("%s: findings %+v, want %q", name, findings, tc.want)
		}
	}
	contract.Postconditions = append(contract.Postconditions, model.Postcondition{ID: "POST", Outcomes: []string{"vanished"}, ViolationBusinessError: "unheard of"})
	if findings := rule.Validate(effectCorpus(t, contract)); len(findings) != 3 {
		t.Errorf("duplicate id, undeclared outcome and undeclared business error: %+v", findings)
	}
}

func TestReconciliationRule(t *testing.T) {
	at := time.Date(2026, 6, 18, 17, 0, 0, 0, time.UTC)
	actor := model.ObjectRef{Kind: model.KindAgentIdentity, ID: "AGENT"}
	attempt := model.ActionRecord{Envelope: envelope(model.KindActionRecord, "ACT"), Actor: actor, CapabilityID: model.Ref{ID: "CAP"}, ActionTime: at, Disposition: model.DispositionUnknown}
	reconciling := attempt
	reconciling.ID, reconciling.ActionTime, reconciling.Disposition, reconciling.ReconcilesActionID = "ACT-2", at.Add(time.Minute), model.DispositionExecuted, &model.Ref{ID: "ACT"}
	rule := ReconciliationRule{AbstractRule{"CHR-RULE-EVID-002", nil}}
	amended := func(edit func(attempt, reconciling *model.ActionRecord)) []Finding {
		a, r := attempt, reconciling
		edit(&a, &r)
		return rule.Validate(effectCorpus(t, a, r))
	}
	if findings := amended(func(_, _ *model.ActionRecord) {}); len(findings) != 0 {
		t.Errorf("reconciled unknown attempt: %+v", findings)
	}
	for name, edit := range map[string]func(a, r *model.ActionRecord){
		"settled attempt":    func(a, _ *model.ActionRecord) { a.Disposition = model.DispositionFailed },
		"another actor":      func(_, r *model.ActionRecord) { r.Actor.ID = "OTHER" },
		"another capability": func(_, r *model.ActionRecord) { r.CapabilityID.ID = "CAP-2" },
		"precedes":           func(_, r *model.ActionRecord) { r.ActionTime = at.Add(-time.Minute) },
	} {
		if findings := amended(edit); len(findings) != 1 {
			t.Errorf("%s: %+v", name, findings)
		}
	}
}
