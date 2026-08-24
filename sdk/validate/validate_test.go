package validate

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

const harbor = "../../examples/harbor-manufacturing/instances"

func TestHarborIsStructurallyValid(t *testing.T) {
	s, err := NewStructural()
	if err != nil {
		t.Fatal(err)
	}
	c, err := corpus.NewDirLoader(harbor).Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Len() < 80 {
		t.Fatalf("expected the Harbor corpus, got %d documents", c.Len())
	}
	for _, f := range s.Validate(c) {
		t.Errorf("%s %s: %s", f.DocumentID, f.Path, f.Message)
	}
}

func TestIdentityStateUsesActionTime(t *testing.T) {
	raw := []byte(`{"lifecycleState":"suspended","lifecycleHistory":[{"state":"active","effectiveAt":"2026-01-01T00:00:00Z"},{"state":"suspended","effectiveAt":"2026-07-01T00:00:00Z"}]}`)
	d := &model.Document{Raw: raw}
	state, err := identityStateAt(d, time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC))
	if err != nil || state != model.LifecycleActive {
		t.Fatalf("historical action state = %q, %v", state, err)
	}
}

func TestIdentityStateRequiresHistoryForActingIdentity(t *testing.T) {
	d := &model.Document{Raw: []byte(`{"lifecycleState":"active"}`)}
	if _, err := identityStateAt(d, time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("acting identity without lifecycleHistory was accepted")
	}
}

func TestApprovalMustBindEvaluatedActionInputsAndSubjects(t *testing.T) {
	c := corpus.New()
	for _, raw := range []string{
		`{"charterSpecVersion":"draft","namespace":"test.example","kind":"Approval","id":"APPR-1","approvedAction":"CAP-1","materialInputsDigest":"sha256:approved","subjectRefs":[{"kind":"ProcessInstance","id":"PROC-1"}],"issuedAt":"2026-01-01T00:00:00Z","validityMode":"expires-at","expiresAt":"2026-12-31T00:00:00Z"}`,
		`{"charterSpecVersion":"draft","namespace":"test.example","kind":"ActionRecord","id":"ACT-1","actor":{"kind":"HumanIdentity","id":"H-1"},"capabilityId":"CAP-1","materialInputsDigest":"sha256:changed","subjectRefs":[{"kind":"ProcessInstance","id":"PROC-2"}],"actionTime":"2026-06-01T00:00:00Z","approvalEvaluations":[{"approvalId":"APPR-1","approvedAction":"CAP-2","result":"approved","reason":"test"}]}`,
	} {
		d, err := (corpus.Parser{}).Parse([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if err := c.Add(d); err != nil {
			t.Fatal(err)
		}
	}
	rule := ApprovalValidRule{AbstractRule{"test", nil}}
	findings := rule.Validate(c)
	if len(findings) != 3 {
		t.Fatalf("got %d findings, want action, input, and subject mismatches: %+v", len(findings), findings)
	}
}

func TestReferencesRespectNamespacesAndExplicitExternal(t *testing.T) {
	meta, err := NewSchemaMeta()
	if err != nil {
		t.Fatal(err)
	}
	rule := ReferencesResolveRule{AbstractRule{"test", nil}, meta}
	c := corpus.New()
	for _, namespace := range []string{"one.example", "two.example"} {
		if err := c.Add(&model.Document{Envelope: model.Envelope{Namespace: namespace, ID: "E", Kind: model.KindEnterprise}}); err != nil {
			t.Fatal(err)
		}
	}
	value := map[string]any{
		"enterpriseId": map[string]any{"namespace": "two.example", "id": "E"},
		"target":       map[string]any{"kind": "VendorObject", "id": "X", "external": true},
		"extensions":   map[string]any{"vendor.example": map[string]any{"kind": "Enterprise", "id": "MISSING"}},
	}
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Add(&model.Document{Envelope: model.Envelope{Namespace: "one.example", ID: "OWNER", Kind: model.KindRole}, Raw: raw, Value: value}); err != nil {
		t.Fatal(err)
	}
	if findings := rule.Validate(c); len(findings) != 0 {
		t.Fatalf("valid cross-namespace/external references failed: %+v", findings)
	}
	value["target"] = map[string]any{"kind": "VendorObject", "id": "X"}
	if findings := rule.Validate(c); len(findings) != 1 {
		t.Fatalf("got %d findings, want explicit-external failure", len(findings))
	}
}

func TestHarborPassesActiveRules(t *testing.T) {
	c, err := corpus.NewDirLoader(harbor).Load()
	if err != nil {
		t.Fatal(err)
	}
	rules, err := NewRuleSet()
	if err != nil {
		t.Fatal(err)
	}
	for id, r := range rules {
		for _, f := range r.Validate(c) {
			t.Errorf("%s: %s: %s", id, f.DocumentID, f.Message)
		}
	}
}

func TestManifestFixtures(t *testing.T) {
	r, err := NewRunner("../../conformance")
	if err != nil {
		t.Fatal(err)
	}
	results, err := r.RunManifest()
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 9 {
		t.Fatalf("expected the Harbor fixture plus one invalid fixture per rule, got %d", len(results))
	}
	for _, res := range results {
		if !res.Passed {
			t.Errorf("%s (%s): %s", res.Dir, res.Expected, res.Detail)
			for _, f := range res.Findings {
				t.Logf("  %s %s: %s", f.RuleID, f.DocumentID, f.Message)
			}
		}
	}
}

func TestStructuralRejectsUnknownProperty(t *testing.T) {
	s, err := NewStructural()
	if err != nil {
		t.Fatal(err)
	}
	d, err := (corpus.Parser{}).Parse([]byte(`{"charterSpecVersion":"draft","namespace":"t","kind":"Enterprise","id":"E1","name":"x","plantId":7}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.ValidateDocument(d)) == 0 {
		t.Fatal("unevaluated property was accepted")
	}
}
