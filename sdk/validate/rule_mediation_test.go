package validate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func mediationDocument(t *testing.T, c *corpus.Corpus, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	d, err := (corpus.Parser{}).Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Add(d); err != nil {
		t.Fatal(err)
	}
}

func TestMediationAttributeTrustMatchesProfile(t *testing.T) {
	profile := model.MediationProfile{
		Envelope:       model.Envelope{Namespace: "test.example", ID: "P", Kind: model.KindMediationProfile},
		AttributeTrust: []model.AttributeTrust{{Attribute: "client", Provenance: model.ProvenanceAsserted, PermittedEffect: model.PermittedTightenOnly}},
	}
	decision := model.MediationDecision{
		Envelope:           model.Envelope{Namespace: "test.example", ID: "D", Kind: model.KindMediationDecision},
		MediationProfileID: model.Ref{ID: "P"},
		DecisionAttributes: []model.DecisionAttribute{{Attribute: "client", Provenance: model.ProvenanceAuthenticated, Effect: model.EffectNone}},
	}
	c := corpus.New()
	mediationDocument(t, c, profile)
	mediationDocument(t, c, decision)
	rule := AttributeProvenanceRule{AbstractRule{"CHR-RULE-MED-003", nil}}
	if findings := rule.Validate(c); len(findings) != 1 {
		t.Fatalf("mismatched provenance: got %d findings, want 1: %+v", len(findings), findings)
	}
	profile.AttributeTrust = append(profile.AttributeTrust, model.AttributeTrust{Attribute: "client", Provenance: model.ProvenanceAuthenticated, PermittedEffect: model.PermittedRelaxOrTighten})
	c = corpus.New()
	mediationDocument(t, c, profile)
	mediationDocument(t, c, decision)
	if findings := rule.Validate(c); len(findings) != 1 {
		t.Fatalf("ambiguous trust declaration: got %d findings, want 1: %+v", len(findings), findings)
	}
}

func TestMediationDecisionIdentifiesScopedSubject(t *testing.T) {
	profile := model.MediationProfile{
		Envelope: model.Envelope{Namespace: "test.example", ID: "P", Kind: model.KindMediationProfile},
		Stage:    model.StageModelInvocation,
		Scope: model.MediationScope{
			AgentIdentityIDs: []model.Ref{{ID: "A"}},
			InformationIDs:   []model.Ref{{ID: "I"}},
		},
		DeclaredOutcomes: []string{model.OutcomeAllow},
	}
	decision := model.MediationDecision{
		Envelope:           model.Envelope{Namespace: "test.example", ID: "D", Kind: model.KindMediationDecision},
		MediationProfileID: model.Ref{ID: "P"},
		Actor:              model.ObjectRef{Kind: model.KindAgentIdentity, ID: "A"},
		Subject: model.MediationSubject{
			CapabilityIDs:  []model.Ref{},
			InformationIDs: []model.Ref{{ID: "I"}},
		},
		Stage:   model.StageModelInvocation,
		Outcome: model.OutcomeAllow,
	}
	rule := DeclaredOutcomeRule{AbstractRule{"CHR-RULE-MED-001", nil}}
	c := corpus.New()
	mediationDocument(t, c, profile)
	mediationDocument(t, c, decision)
	if findings := rule.Validate(c); len(findings) != 0 {
		t.Fatalf("scoped subject rejected: %+v", findings)
	}
	decision.Subject.InformationIDs = []model.Ref{}
	c = corpus.New()
	mediationDocument(t, c, profile)
	mediationDocument(t, c, decision)
	if findings := rule.Validate(c); len(findings) != 1 {
		t.Fatalf("missing scoped information: got %d findings, want 1: %+v", len(findings), findings)
	}
}

func TestRepeatedMediationSubmissionRequiresSupersession(t *testing.T) {
	first := model.MediationDecision{
		Envelope:     model.Envelope{Namespace: "test.example", ID: "D1", Kind: model.KindMediationDecision},
		SubmissionID: "S1", Outcome: model.OutcomeAllow, DecidedAt: time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC),
	}
	second := first
	second.ID, second.Outcome = "D2", model.OutcomeDeny
	second.DecidedAt = first.DecidedAt.Add(time.Second)
	rule := RepeatedSubmissionRule{AbstractRule{"CHR-RULE-MED-004", nil}}
	c := corpus.New()
	mediationDocument(t, c, first)
	mediationDocument(t, c, second)
	if findings := rule.Validate(c); len(findings) != 1 {
		t.Fatalf("contradictory decisions: got %d findings, want 1: %+v", len(findings), findings)
	}
	second.Supersedes = &model.Ref{ID: first.ID}
	c = corpus.New()
	mediationDocument(t, c, first)
	mediationDocument(t, c, second)
	if findings := rule.Validate(c); len(findings) != 0 {
		t.Fatalf("explicit supersession rejected: %+v", findings)
	}
	third := second
	third.ID, third.Outcome = "D3", model.OutcomeModify
	third.DecidedAt = second.DecidedAt.Add(time.Second)
	third.Supersedes = &model.Ref{ID: second.ID}
	mediationDocument(t, c, third)
	if findings := rule.Validate(c); len(findings) != 0 {
		t.Fatalf("supersession chain rejected: %+v", findings)
	}
	third.DecidedAt = second.DecidedAt
	c = corpus.New()
	mediationDocument(t, c, first)
	mediationDocument(t, c, second)
	mediationDocument(t, c, third)
	if findings := rule.Validate(c); len(findings) == 0 {
		t.Fatal("equal-timestamp supersession passed")
	}
}

func TestLayeredMediatorsMayDisagreeOnOneSubmission(t *testing.T) {
	first := model.MediationDecision{
		Envelope:           model.Envelope{Namespace: "test.example", ID: "D1", Kind: model.KindMediationDecision},
		MediationProfileID: model.Ref{ID: "ENTERPRISE"},
		SubmissionID:       "S1",
		Outcome:            model.OutcomeAllow,
	}
	second := first
	second.ID, second.MediationProfileID, second.Outcome = "D2", model.Ref{ID: "PLATFORM"}, model.OutcomeDeny
	c := corpus.New()
	mediationDocument(t, c, first)
	mediationDocument(t, c, second)
	rule := RepeatedSubmissionRule{AbstractRule{"CHR-RULE-MED-004", nil}}
	if findings := rule.Validate(c); len(findings) != 0 {
		t.Fatalf("independent mediators were treated as contradictory: %+v", findings)
	}
}

func TestMediationSupersessionStaysWithinSubmissionAndProfile(t *testing.T) {
	first := model.MediationDecision{
		Envelope:           model.Envelope{Namespace: "test.example", ID: "D1", Kind: model.KindMediationDecision},
		MediationProfileID: model.Ref{ID: "P1"},
		SubmissionID:       "S1",
		Outcome:            model.OutcomeAllow,
		DecidedAt:          time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC),
	}
	second := first
	second.ID, second.SubmissionID = "D2", "S2"
	second.DecidedAt = first.DecidedAt.Add(time.Second)
	second.Supersedes = &model.Ref{ID: first.ID}
	rule := RepeatedSubmissionRule{AbstractRule{"CHR-RULE-MED-004", nil}}
	c := corpus.New()
	mediationDocument(t, c, first)
	mediationDocument(t, c, second)
	if findings := rule.Validate(c); len(findings) != 1 {
		t.Fatalf("foreign submission supersession: got %d findings, want 1: %+v", len(findings), findings)
	}
	second.SubmissionID = first.SubmissionID
	second.MediationProfileID = model.Ref{ID: "P2"}
	c = corpus.New()
	mediationDocument(t, c, first)
	mediationDocument(t, c, second)
	if findings := rule.Validate(c); len(findings) != 1 {
		t.Fatalf("foreign profile supersession: got %d findings, want 1: %+v", len(findings), findings)
	}
}

func TestMediationProfileRejectsObsoleteEvidenceFlag(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(harbor, "mediation", "MEDPROF-MODEL-INVOCATION.json"))
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	value["evidenceObligation"].(map[string]any)["recordsDecision"] = false
	raw, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	d, err := (corpus.Parser{}).Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStructural()
	if err != nil {
		t.Fatal(err)
	}
	if findings := s.ValidateDocument(d); len(findings) == 0 {
		t.Fatal("profile with obsolete recordsDecision field passed structural validation")
	}
}

func TestMediationDecisionRequiresSubjectContext(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(harbor, "mediation", "MEDDEC-OE-0042-01.json"))
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	delete(value, "subject")
	raw, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	d, err := (corpus.Parser{}).Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStructural()
	if err != nil {
		t.Fatal(err)
	}
	if findings := s.ValidateDocument(d); len(findings) == 0 {
		t.Fatal("decision without submitted-act subject passed structural validation")
	}
}

func TestShadowProfileDecisionIsNotEnforced(t *testing.T) {
	profile := model.MediationProfile{
		Envelope:         model.Envelope{Namespace: "test.example", ID: "P", Kind: model.KindMediationProfile},
		DeclaredOutcomes: []string{model.OutcomeAllow, model.OutcomeDeny},
		EnforcementMode:  model.EnforcementShadow, FailurePolicy: model.FailureObserve,
	}
	decision := model.MediationDecision{
		Envelope:           model.Envelope{Namespace: "test.example", ID: "D", Kind: model.KindMediationDecision},
		MediationProfileID: model.Ref{ID: "P"}, Outcome: model.OutcomeDeny,
		MediatorAvailability: model.MediatorAvailable, Enforced: true,
	}
	rule := UnavailableMediatorRule{AbstractRule{"CHR-RULE-MED-002", nil}}
	c := corpus.New()
	mediationDocument(t, c, profile)
	mediationDocument(t, c, decision)
	if findings := rule.Validate(c); len(findings) != 1 {
		t.Fatalf("enforced deny under shadow profile: got %d findings, want 1: %+v", len(findings), findings)
	}
	decision.Enforced = false
	c = corpus.New()
	mediationDocument(t, c, profile)
	mediationDocument(t, c, decision)
	if findings := rule.Validate(c); len(findings) != 0 {
		t.Fatalf("unenforced deny under shadow profile rejected: %+v", findings)
	}
}

func TestDenyMayCarryDecisionHandlingObligation(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(harbor, "mediation", "MEDDEC-OE-0042-01.json"))
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	value["obligationsApplied"] = []map[string]any{{"obligation": "notify", "appliedTo": "owner", "targetKind": "decision", "authorityExpanded": false}}
	raw, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	d, err := (corpus.Parser{}).Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStructural()
	if err != nil {
		t.Fatal(err)
	}
	if findings := s.ValidateDocument(d); len(findings) != 0 {
		t.Fatalf("deny with notification rejected: %+v", findings)
	}
}

func TestMediationDispositionStructure(t *testing.T) {
	s, err := NewStructural()
	if err != nil {
		t.Fatal(err)
	}
	load := func(name string) map[string]any {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(harbor, "mediation", name))
		if err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err := json.Unmarshal(raw, &value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	check := func(name string, value map[string]any, valid bool) {
		t.Helper()
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		d, err := (corpus.Parser{}).Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		findings := s.ValidateDocument(d)
		if (len(findings) == 0) != valid {
			t.Fatalf("%s: findings = %+v, want valid = %t", name, findings, valid)
		}
	}
	shadow := load("MEDDEC-OE-0042-02.json")
	check("shadow proposal", shadow, true)
	shadow["obligationsApplied"] = shadow["obligationsProposed"]
	check("shadow act alteration", shadow, false)
	delete(shadow, "obligationsApplied")
	shadow["stage"] = "host:planning"
	check("binding stage", shadow, true)

	ask := load("MEDDEC-OE-0042-01.json")
	ask["outcome"] = model.OutcomeAsk
	check("ask without handoff", ask, false)
	ask["handoff"] = map[string]any{"destination": "human-review", "request": "Approve this act."}
	check("ask with handoff", ask, true)
	delete(ask, "subjectReason")
	check("ask without reason", ask, false)
}
