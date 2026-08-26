package validate

import (
	"encoding/json"
	"testing"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func TestClaimIsSchemaValidAndConforming(t *testing.T) {
	r, err := NewRunner("../../conformance")
	if err != nil {
		t.Fatal(err)
	}
	claim, err := r.Claim(ClaimSpec{Namespace: "sdk.example", ID: "CLAIM-CORE-MODEL", Name: "Reference SDK core-model claim", Profile: "core-model",
		Implementation: model.ObjectRef{Kind: "GoModule", ID: "github.com-nauticana-charter", External: true}, ImplementationVersion: "1.0.0", ResultDate: "2026-08-25"})
	if err != nil {
		t.Fatal(err)
	}
	m, err := r.Manifest.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	if claim.Result != model.ResultConforming || len(claim.TestedRuleIDs) != len(m.Profiles[0].Rules) || len(claim.Results) != len(m.Profiles[0].Rules) {
		t.Fatalf("claim: %+v", claim)
	}
	raw, err := json.Marshal(claim)
	if err != nil {
		t.Fatal(err)
	}
	d, err := (corpus.Parser{}).Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range r.Structural.ValidateDocument(d) {
		t.Errorf("%s: %s", f.Path, f.Message)
	}
	if _, err := r.Claim(ClaimSpec{Profile: "vendor-pack"}); err == nil {
		t.Error("unknown profile must not yield a claim")
	}
}

// TestReleasedClaimsAreCurrent fails when the committed reference claims fall behind the manifest's release.
func TestReleasedClaimsAreCurrent(t *testing.T) {
	r, err := NewRunner("../../conformance")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := r.Manifest.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	c, err := corpus.NewDirLoader("../../conformance/claims").Load()
	if err != nil {
		t.Fatal(err)
	}
	claimed := map[string]bool{}
	for _, d := range c.Documents() {
		for _, f := range r.Structural.ValidateDocument(d) {
			t.Errorf("%s: %s %s", d.ID, f.Path, f.Message)
		}
		claim, err := corpus.Decode[model.ConformanceClaim](d)
		if err != nil {
			t.Fatal(err)
		}
		if claim.CharterSpecVersion != manifest.Specification.Version || claim.Result != model.ResultConforming {
			t.Errorf("%s: version %s result %s, want %s conforming", d.ID, claim.CharterSpecVersion, claim.Result, manifest.Specification.Version)
		}
		for _, p := range manifest.Profiles {
			if p.ID == claim.ConformanceProfile && len(claim.TestedRuleIDs) != len(p.Rules) {
				t.Errorf("%s: %d tested rules, profile has %d", d.ID, len(claim.TestedRuleIDs), len(p.Rules))
			}
		}
		claimed[claim.ConformanceProfile] = true
	}
	for _, p := range manifest.Profiles {
		if p.Status == "active" && !claimed[p.ID] {
			t.Errorf("active profile %s has no committed reference claim", p.ID)
		}
	}
}
