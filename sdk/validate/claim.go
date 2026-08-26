package validate

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/nauticana/charter/sdk/model"
)

// ClaimSpec identifies the implementation, profile, and document identity of a generated conformance claim.
type ClaimSpec struct {
	Namespace             string
	ID                    string
	Name                  string
	Profile               string
	Implementation        model.ObjectRef
	ImplementationVersion string
	ResultDate            model.Date
}

// Claim runs the manifest fixtures and generates a ConformanceClaim for one active profile: a rule passes only when both of
// its fixtures pass, and the claim is partial when any profile rule could not be tested (CHR-CONF-005, CHR-CONF-009, CHR-CONF-011).
func (r *Runner) Claim(spec ClaimSpec) (model.ConformanceClaim, error) {
	m, err := r.Manifest.Manifest()
	if err != nil {
		return model.ConformanceClaim{}, err
	}
	var profileRules []string
	found := false
	for _, p := range m.Profiles {
		if p.ID == spec.Profile {
			if p.Status != "active" {
				return model.ConformanceClaim{}, fmt.Errorf("profile %s is %s", p.ID, p.Status)
			}
			profileRules, found = p.Rules, true
		}
	}
	if !found {
		return model.ConformanceClaim{}, fmt.Errorf("unknown profile %s", spec.Profile)
	}
	results, err := r.RunManifest()
	if err != nil {
		return model.ConformanceClaim{}, err
	}
	byDir := map[string]FixtureResult{}
	for _, res := range results {
		byDir[res.Dir] = res
	}
	paths := map[string]string{}
	for _, entry := range m.Rules {
		paths[entry.ID] = entry.Path
	}
	claim := model.ConformanceClaim{Implementation: spec.Implementation}
	claim.CharterSpecVersion, claim.Namespace, claim.ID, claim.Kind, claim.Name = m.Specification.Version, spec.Namespace, spec.ID, model.KindConformanceClaim, spec.Name
	claim.ConformanceProfile, claim.ImplementationVersion, claim.ResultDate = spec.Profile, spec.ImplementationVersion, spec.ResultDate
	verifications := map[string]bool{model.VerificationStructural: true}
	claim.Result = model.ResultConforming
	for _, id := range profileRules {
		result := model.RuleResult{RuleID: id, VerificationType: model.VerificationSemantic, Result: model.RuleNotTested}
		def, err := r.Manifest.RuleDefinition(paths[id])
		switch {
		case err != nil:
			result.Details = "rule definition unavailable"
		case def.Verification == string(ClassBehavioral) && !r.subjects().Has(r.Behavioral[id].Kind):
			result.VerificationType, result.Details = def.Verification, fmt.Sprintf("no %s under test", r.Behavioral[id].Kind)
		default:
			result.VerificationType = def.Verification
			var failures []string
			for _, dir := range []string{def.Fixtures.Valid, def.Fixtures.Invalid} {
				res, ok := byDir[filepath.Join(r.Manifest.Dir, dir)]
				if !ok {
					failures = append(failures, dir+": not executed")
				} else if !res.Passed {
					failures = append(failures, dir+": "+res.Detail)
				}
			}
			result.Result = model.RulePass
			if len(failures) > 0 {
				result.Result, result.Details = model.RuleFail, strings.Join(failures, "; ")
			}
		}
		switch result.Result {
		case model.RulePass:
			claim.TestedRuleIDs = append(claim.TestedRuleIDs, id)
			verifications[result.VerificationType] = true
		case model.RuleFail:
			claim.TestedRuleIDs = append(claim.TestedRuleIDs, id)
			verifications[result.VerificationType] = true
			claim.Result = model.ResultNonconforming
		default:
			if claim.Result == model.ResultConforming {
				claim.Result = model.ResultPartial
			}
		}
		claim.Results = append(claim.Results, result)
	}
	for _, v := range []string{model.VerificationStructural, model.VerificationSemantic, model.VerificationBehavioral} {
		if verifications[v] {
			claim.VerificationTypes = append(claim.VerificationTypes, v)
		}
	}
	slices.Sort(claim.TestedRuleIDs)
	return claim, nil
}
