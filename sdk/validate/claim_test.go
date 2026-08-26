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
		Implementation: model.ObjectRef{Kind: "GoModule", ID: "github.com-nauticana-charter", External: true}, ImplementationVersion: "0.1.0-draft", ResultDate: "2026-08-25"})
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
