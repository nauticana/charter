package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/validate"
)

const (
	harbor = "../../examples/harbor-manufacturing/instances"
	conf   = "../../conformance"
)

func exec(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestValidateHarbor(t *testing.T) {
	code, out, errOut := exec(t, "validate", "-format", "json", "-conformance", conf, harbor)
	if code != exitOK {
		t.Fatalf("exit %d: %s%s", code, out, errOut)
	}
	var r validationReport
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatal(err)
	}
	if r.Result != model.ResultConforming || r.Documents < 80 || r.Profile != "core-model" || len(r.Rules) < 19 || r.SpecVersion != "1.0.0" {
		t.Errorf("report: %+v", r)
	}
	if code, out, _ := exec(t, "validate", harbor); code != exitOK || !strings.Contains(out, "result: conforming") {
		t.Errorf("text report: %d %s", code, out)
	}
}

func TestValidateReportsFindingsWithRequirements(t *testing.T) {
	code, out, _ := exec(t, "validate", "-format", "json", conf+"/fixtures/invalid/expired-grant")
	if code != exitFailed {
		t.Fatalf("exit %d: %s", code, out)
	}
	var r validationReport
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range r.Findings {
		if f.RuleID == "CHR-RULE-AUTH-001" && len(f.Requirements) == 2 && f.DocumentID == "ACT-FX" {
			found = true
		}
	}
	if !found || r.Result != model.ResultNonconforming {
		t.Errorf("findings: %+v", r.Findings)
	}
	code, out, _ = exec(t, "validate", "-spec-version", "2.0.0", harbor)
	if code != exitFailed || !strings.Contains(out, "spec-version") || !strings.Contains(out, "CHR-CONF-001") || !strings.Contains(out, "semantic rules skipped") {
		t.Errorf("spec version mismatch: %d %s", code, out)
	}
}

func TestValidateUsageAndErrors(t *testing.T) {
	if code, _, _ := exec(t, "validate"); code != exitUsage {
		t.Error("missing directory accepted")
	}
	if code, _, errOut := exec(t, "validate", "-conformance", conf, "-profile", "vendor-pack", harbor); code != exitUsage || !strings.Contains(errOut, "unknown profile") {
		t.Errorf("unknown profile: %d %s", code, errOut)
	}
	if code, out, _ := exec(t, "validate", "-conformance", conf, "-profile", "agent-runtime", harbor); code != exitOK || !strings.Contains(out, "0 rules") {
		t.Errorf("behavioral-only profile must run no document rules: %d %s", code, out)
	}
	if code, _, _ := exec(t, "validate", "-format", "yaml", harbor); code != exitUsage {
		t.Error("unknown format accepted")
	}
	if code, _, _ := exec(t, "frobnicate"); code != exitUsage {
		t.Error("unknown command accepted")
	}
	if code, out, _ := exec(t, "help"); code != exitOK || !strings.Contains(out, "usage: charter") {
		t.Error("help")
	}
}

func TestConformanceAndClaim(t *testing.T) {
	code, out, errOut := exec(t, "conformance", "-format", "json", "-conformance", conf)
	if code != exitOK {
		t.Fatalf("exit %d: %s%s", code, out, errOut)
	}
	var r conformanceReport
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatal(err)
	}
	if r.Result != model.ResultConforming || len(r.Fixtures) < 20 {
		t.Errorf("conformance: %+v", r)
	}
	if code, out, _ := exec(t, "conformance", "-conformance", conf); code != exitOK || !strings.Contains(out, "PASS  fail") {
		t.Errorf("text conformance: %d %s", code, out)
	}
	code, out, errOut = exec(t, "claim", "-conformance", conf, "-namespace", "sdk.example", "-id", "CLAIM-1", "-implementation", "github.com-nauticana-charter", "-implementation-version", "1.0.0", "-date", "2026-08-25")
	if code != exitOK {
		t.Fatalf("claim exit %d: %s%s", code, out, errOut)
	}
	d, err := (corpus.Parser{}).Parse([]byte(out))
	if err != nil || d.Kind != model.KindConformanceClaim {
		t.Fatalf("claim output: %v %v", d, err)
	}
	structural, err := validate.NewStructural()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range structural.ValidateDocument(d) {
		t.Errorf("claim %s: %s", f.Path, f.Message)
	}
	if code, _, _ := exec(t, "claim", "-conformance", conf); code != exitUsage {
		t.Error("claim without identity accepted")
	}
	if code, out, _ := exec(t, "conformance", "-conformance", conf, "-subject", "none"); code != exitOK || strings.Contains(out, "governed-execution") {
		t.Errorf("conformance without subject: %d %s", code, out)
	}
	code, out, _ = exec(t, "claim", "-conformance", conf, "-subject", "none", "-profile", "agent-runtime", "-namespace", "sdk.example", "-id", "CLAIM-2", "-implementation", "x", "-implementation-version", "0")
	if code != exitFailed || !strings.Contains(out, `"result": "partial"`) {
		t.Errorf("behavioral claim without subject must be partial: %d %s", code, out)
	}
	if code, _, _ := exec(t, "conformance", "-subject", "vendor"); code != exitUsage {
		t.Error("unknown subject accepted")
	}
}

func TestVersion(t *testing.T) {
	code, out, _ := exec(t, "version", "-conformance", conf)
	if code != exitOK || !strings.Contains(out, "specification 1.0.0") || !strings.Contains(out, "active profiles core-model") {
		t.Errorf("version: %d %s", code, out)
	}
	code, out, _ = exec(t, "version", "-format", "json")
	var v versionReport
	if code != exitOK || json.Unmarshal([]byte(out), &v) != nil || len(v.ImplementedRules) < 19 || v.CatalogVersion == "" {
		t.Errorf("version json: %d %s", code, out)
	}
}
