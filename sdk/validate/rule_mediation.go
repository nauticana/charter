package validate

import (
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// DeclaredOutcomeRule checks each decision's stage, subject, and outcome against its profile.
type DeclaredOutcomeRule struct{ AbstractRule }

var _ Rule = DeclaredOutcomeRule{}

func (r DeclaredOutcomeRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, d := range allOf[model.MediationDecision](c, model.KindMediationDecision) {
		profile, ok := resolveAs[model.MediationProfile](c, d.Namespace, d.MediationProfileID, model.KindMediationProfile)
		if !ok {
			continue
		}
		if d.Stage != profile.Stage {
			out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("decides at stage %s; profile %s mediates %s", d.Stage, profile.ID, profile.Stage)))
		}
		if len(profile.Scope.AgentIdentityIDs) > 0 && (d.Actor.Kind != model.KindAgentIdentity || !mediationRefInScope(d.Namespace, model.Ref{Namespace: d.Actor.Namespace, ID: d.Actor.ID}, profile.Scope.AgentIdentityIDs)) {
			out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("actor is outside profile %s agent scope", profile.ID)))
		}
		for _, dimension := range []struct {
			name     string
			subject  []model.Ref
			declared []model.Ref
		}{
			{"capability", d.Subject.CapabilityIDs, profile.Scope.CapabilityIDs},
			{"information", d.Subject.InformationIDs, profile.Scope.InformationIDs},
		} {
			if len(dimension.declared) > 0 && len(dimension.subject) == 0 {
				out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("subject identifies no %s in profile %s scope", dimension.name, profile.ID)))
			}
			for _, ref := range dimension.subject {
				if len(dimension.declared) > 0 && !mediationRefInScope(d.Namespace, ref, dimension.declared) {
					out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("subject %s %s is outside profile %s scope", dimension.name, ref.ID, profile.ID)))
				}
			}
		}
		if d.Outcome == model.OutcomeNotDecided {
			continue
		}
		if !profile.Declares(d.Outcome) {
			out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("outcome %s is not declared by profile %s", d.Outcome, profile.ID)))
		}
	}
	return out
}

func mediationRefInScope(namespace string, ref model.Ref, scope []model.Ref) bool {
	key := corpus.KeyOf(namespace, ref)
	for _, candidate := range scope {
		if corpus.KeyOf(namespace, candidate) == key {
			return true
		}
	}
	return false
}

// UnavailableMediatorRule checks that an act the mediator did not decide is never evidenced as an allow, that the
// declared failure policy is the one applied, that an undecided act states why, and that a decision is enforced
// exactly when its profile enforces (CHR-MED-002, CHR-MED-005, CHR-MED-006).
type UnavailableMediatorRule struct{ AbstractRule }

var _ Rule = UnavailableMediatorRule{}

func (r UnavailableMediatorRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, d := range allOf[model.MediationDecision](c, model.KindMediationDecision) {
		decided := d.MediatorAvailability == model.MediatorAvailable
		if !decided && d.Outcome == model.OutcomeAllow {
			out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("mediator was %s yet the act is recorded as allowed", d.MediatorAvailability)))
		}
		profile, ok := resolveAs[model.MediationProfile](c, d.Namespace, d.MediationProfileID, model.KindMediationProfile)
		if !ok {
			continue
		}
		if !decided && d.AppliedFailurePolicy != profile.FailurePolicy {
			out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("applied failure policy %q differs from the %q declared by profile %s", d.AppliedFailurePolicy, profile.FailurePolicy, profile.ID)))
		}
		if !decided && profile.FailurePolicy == model.FailureFailClosed && d.Outcome != model.OutcomeDeny {
			out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("profile %s fails closed; an undecided act must be denied, not %s", profile.ID, d.Outcome)))
		}
		if !decided && profile.FailurePolicy != model.FailureFailClosed && d.Outcome != model.OutcomeNotDecided {
			out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("profile %s applies %s; an undecided act must be recorded as not-decided, not %s", profile.ID, profile.FailurePolicy, d.Outcome)))
		}
		if wantEnforced := profile.EnforcementMode == model.EnforcementEnforce && d.Outcome != model.OutcomeNotDecided; d.Enforced != wantEnforced {
			out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("profile %s is in %s mode, so enforced must be %t", profile.ID, profile.EnforcementMode, wantEnforced)))
		}
		if d.Outcome == model.OutcomeNotDecided && profile.Coverage.SubmittedShare == 1 && d.NotDecidedReason == "not-sampled" {
			out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("profile %s submits every act, so not-sampled cannot explain an undecided act", profile.ID)))
		}
	}
	return out
}

// AttributeProvenanceRule checks that no decision was relaxed by an attribute the profile declares as asserted or
// tighten-only, and that every attribute a decision rests on is one the profile declares (CHR-MED-007).
type AttributeProvenanceRule struct{ AbstractRule }

var _ Rule = AttributeProvenanceRule{}

func (r AttributeProvenanceRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, profile := range allOf[model.MediationProfile](c, model.KindMediationProfile) {
		seen := map[string]bool{}
		for _, attribute := range profile.AttributeTrust {
			if seen[attribute.Attribute] {
				out = append(out, r.finding(profile.Namespace, profile.ID, fmt.Sprintf("attribute %q has more than one trust declaration", attribute.Attribute)))
			}
			seen[attribute.Attribute] = true
		}
	}
	for _, d := range allOf[model.MediationDecision](c, model.KindMediationDecision) {
		profile, ok := resolveAs[model.MediationProfile](c, d.Namespace, d.MediationProfileID, model.KindMediationProfile)
		if !ok {
			continue
		}
		declared := map[string]model.AttributeTrust{}
		for _, a := range profile.AttributeTrust {
			declared[a.Attribute] = a
		}
		for _, used := range d.DecisionAttributes {
			trust, known := declared[used.Attribute]
			if !known {
				out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("attribute %q is not declared by profile %s", used.Attribute, profile.ID)))
				continue
			}
			if used.Provenance != trust.Provenance {
				out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("attribute %q records provenance %q, but profile %s declares %q", used.Attribute, used.Provenance, profile.ID, trust.Provenance)))
			}
			if used.Effect != model.EffectRelaxed {
				continue
			}
			if trust.Provenance == model.ProvenanceAsserted || trust.PermittedEffect == model.PermittedTightenOnly {
				out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("attribute %q relaxed the decision although profile %s declares it %s/%s", used.Attribute, profile.ID, trust.Provenance, trust.PermittedEffect)))
			}
		}
	}
	return out
}

// RepeatedSubmissionRule rejects contradictory decisions unless the later decision explicitly supersedes the earlier one.
type RepeatedSubmissionRule struct{ AbstractRule }

var _ Rule = RepeatedSubmissionRule{}

func (r RepeatedSubmissionRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	type submissionKey struct {
		Namespace    string
		SubmissionID string
		Profile      corpus.DocumentKey
	}
	bySubmission := map[submissionKey][]model.MediationDecision{}
	for _, d := range allOf[model.MediationDecision](c, model.KindMediationDecision) {
		key := submissionKey{d.Namespace, d.SubmissionID, corpus.KeyOf(d.Namespace, d.MediationProfileID)}
		bySubmission[key] = append(bySubmission[key], d)
		if d.Supersedes == nil {
			continue
		}
		previous, ok := resolveAs[model.MediationDecision](c, d.Namespace, *d.Supersedes, model.KindMediationDecision)
		if !ok {
			continue
		}
		if previous.Namespace != d.Namespace || previous.SubmissionID != d.SubmissionID || corpus.KeyOf(previous.Namespace, previous.MediationProfileID) != key.Profile {
			out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("supersedes decision %s from a different submission or mediation profile", previous.ID)))
		}
		if !d.DecidedAt.After(previous.DecidedAt) {
			out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("supersedes decision %s without a strictly later decidedAt", previous.ID)))
		}
	}
	for _, decisions := range bySubmission {
		byID := map[string]model.MediationDecision{}
		for _, d := range decisions {
			byID[d.ID] = d
		}
		for i, d := range decisions {
			for _, other := range decisions[:i] {
				if d.Outcome == other.Outcome || mediationSupersedes(d, other, byID) || mediationSupersedes(other, d, byID) {
					continue
				}
				out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("submission %s has contradictory outcomes %s and %s without valid supersession", d.SubmissionID, other.Outcome, d.Outcome)))
			}
		}
	}
	return out
}

func mediationSupersedes(later, earlier model.MediationDecision, byID map[string]model.MediationDecision) bool {
	seen := map[string]bool{}
	for later.Supersedes != nil {
		ref := later.Supersedes
		if ref.Namespace != "" && ref.Namespace != later.Namespace {
			return false
		}
		previous, ok := byID[ref.ID]
		if !ok || seen[previous.ID] || !later.DecidedAt.After(previous.DecidedAt) {
			return false
		}
		if previous.ID == earlier.ID {
			return true
		}
		seen[previous.ID] = true
		later = previous
	}
	return false
}
