package binding

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nauticana/charter/sdk/model"
)

const SupportUndeclared = "undeclared"

var ErrUnsupportedFeature = errors.New("binding does not support a required feature")

// Features evaluates declared feature support; an undeclared feature is not supported (CHR-BIND-002, CHR-BIND-009).
type Features []model.FeatureSupport

func (f Features) Support(feature string) string {
	for _, s := range f {
		if s.Feature == feature {
			return s.Support
		}
	}
	return SupportUndeclared
}

// Require fails closed unless every required feature is declared supported (CHR-BIND-006).
func (f Features) Require(required ...string) error {
	var missing []string
	for _, feature := range required {
		if s := f.Support(feature); s != model.SupportSupported {
			missing = append(missing, feature+" ("+s+")")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrUnsupportedFeature, strings.Join(missing, ", "))
	}
	return nil
}

// Partial lists the features declared partial, for partial-support reporting (CHR-CONF-009).
func (f Features) Partial() []string {
	var out []string
	for _, s := range f {
		if s.Support == model.SupportPartial {
			out = append(out, s.Feature)
		}
	}
	return out
}

// Compare evaluates observed support against declared support: a declared feature evaluated below its declaration is
// nonconforming, a partial declaration or evaluation is partial, and unsupported declarations claim nothing (CHR-BIND-009).
func Compare(declared, evaluated []model.FeatureSupport) string {
	observed := Features(evaluated)
	result := model.ResultConforming
	for _, d := range declared {
		if d.Support == model.SupportUnsupported {
			continue
		}
		switch observed.Support(d.Feature) {
		case model.SupportSupported:
			if d.Support == model.SupportPartial {
				result = worse(result, model.ResultPartial)
			}
		case model.SupportPartial, SupportUndeclared:
			result = worse(result, model.ResultPartial)
		default:
			result = worse(result, model.ResultNonconforming)
		}
	}
	return result
}

func worse(current, candidate string) string {
	rank := map[string]int{model.ResultConforming: 0, model.ResultPartial: 1, model.ResultNonconforming: 2}
	if rank[candidate] > rank[current] {
		return candidate
	}
	return current
}

// Conformance records the evaluation of one binding's declared features as a BindingConformance document.
func Conformance(binding model.Envelope, declared, evaluated []model.FeatureSupport, claim model.ConformanceEnvelope) model.BindingConformance {
	claim.Result = Compare(declared, evaluated)
	out := model.BindingConformance{ConformanceEnvelope: claim, BindingID: model.Ref{Namespace: binding.Namespace, ID: binding.ID}, EvaluatedFeatures: evaluated}
	out.Kind = model.KindBindingConformance
	return out
}
