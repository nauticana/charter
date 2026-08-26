package validate

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"sort"

	"github.com/nauticana/charter/sdk/corpus"
)

type FixtureResult struct {
	Dir      string    `json:"dir"`
	Expected string    `json:"expected"`
	Passed   bool      `json:"passed"`
	Detail   string    `json:"detail,omitempty"`
	Findings []Finding `json:"findings,omitempty"`
}

// Runner executes fixtures against the structural validator, the semantic rules, and, through the Subject, the
// runtime-behavioral rules; a nil Subject leaves behavioral rules untested.
type Runner struct {
	Structural *Structural
	Rules      RuleSet
	Behavioral BehavioralRuleSet
	Subject    Subject
	Adapter    AdapterSubject
	Manifest   ManifestReader
}

func (r *Runner) subjects() Subjects { return Subjects{Runtime: r.Subject, Adapter: r.Adapter} }

func NewRunner(confDir string) (*Runner, error) {
	s, err := NewStructural()
	if err != nil {
		return nil, err
	}
	rules, err := NewRuleSet()
	if err != nil {
		return nil, err
	}
	return &Runner{Structural: s, Rules: rules, Behavioral: NewBehavioralRuleSet(), Subject: ReferenceSubject{}, Adapter: ReferenceAdapter{}, Manifest: ManifestReader{Dir: confDir}}, nil
}

// RunFixture evaluates one fixture directory: a valid fixture must produce no findings for its rules; an invalid fixture
// must fail every declared rule and no undeclared rule; a scenario fixture must see the Subject behave as expected.
// Every document must be schema-valid.
func (r *Runner) RunFixture(dir string) (FixtureResult, error) {
	res := FixtureResult{Dir: dir}
	fx, err := r.Manifest.Fixture(dir)
	if err != nil {
		return res, err
	}
	manifest, err := r.Manifest.Manifest()
	if err != nil {
		return res, err
	}
	if fx.SpecificationVersion != manifest.Specification.Version {
		return res, fmt.Errorf("%s: fixture specification version %q does not match manifest %q", dir, fx.SpecificationVersion, manifest.Specification.Version)
	}
	res.Expected = fx.Expected
	docDir := dir
	if fx.Documents != "" {
		docDir = filepath.Join(dir, fx.Documents)
	}
	c, err := corpus.NewDirLoader(docDir).Load()
	if err != nil {
		return res, err
	}
	if sf := r.Structural.Validate(c); len(sf) > 0 {
		res.Findings, res.Detail = sf, "documents are not schema-valid"
		return res, nil
	}
	if fx.Scenario != nil {
		return r.runScenario(res, dir, fx, c)
	}
	declared := map[string]bool{}
	for _, id := range append(append([]string{}, fx.Rules...), fx.AlsoFails...) {
		if _, ok := r.Rules[id]; !ok {
			return res, fmt.Errorf("%s: fixture declares unknown rule %s", dir, id)
		}
		declared[id] = true
	}
	failed := map[string]bool{}
	for id, rule := range r.Rules {
		if f := rule.Validate(c); len(f) > 0 {
			failed[id] = true
			res.Findings = append(res.Findings, f...)
		}
	}
	switch fx.Expected {
	case "pass":
		for _, id := range fx.Rules {
			if failed[id] {
				res.Detail = "valid fixture failed " + id
				return res, nil
			}
		}
	case "fail":
		for id := range declared {
			if !failed[id] {
				res.Detail = "invalid fixture did not fail " + id
				return res, nil
			}
		}
		for id := range failed {
			if !declared[id] {
				res.Detail = "invalid fixture unexpectedly failed " + id
				return res, nil
			}
		}
	default:
		return res, fmt.Errorf("%s: unknown expected value %q", dir, fx.Expected)
	}
	res.Passed = true
	return res, nil
}

func (r *Runner) runScenario(res FixtureResult, dir string, fx *FixtureDescriptor, c *corpus.Corpus) (FixtureResult, error) {
	for _, id := range fx.Rules {
		rule, ok := r.Behavioral[id]
		if !ok {
			return res, fmt.Errorf("%s: fixture declares unknown behavioral rule %s", dir, id)
		}
		if !r.subjects().Has(rule.Kind) {
			return res, fmt.Errorf("%s: scenario fixture needs a %s under test", dir, rule.Kind)
		}
		res.Findings = append(res.Findings, rule.Run(context.Background(), r.subjects(), c, fx.Scenario)...)
	}
	if len(res.Findings) > 0 {
		res.Detail = "subject deviated from the scenario"
		return res, nil
	}
	res.Passed = true
	return res, nil
}

// RunManifest executes every fixture referenced by the active rules of the manifest; behavioral fixtures run only with a Subject.
func (r *Runner) RunManifest() ([]FixtureResult, error) {
	m, err := r.Manifest.Manifest()
	if err != nil {
		return nil, err
	}
	activeProfiles := map[string]map[string]bool{}
	seenProfiles := map[string]bool{}
	for _, profile := range m.Profiles {
		if seenProfiles[profile.ID] {
			return nil, fmt.Errorf("duplicate profile %s", profile.ID)
		}
		seenProfiles[profile.ID] = true
		if profile.Status != "active" {
			continue
		}
		activeProfiles[profile.ID] = map[string]bool{}
		for _, id := range profile.Rules {
			activeProfiles[profile.ID][id] = true
		}
	}
	dirs := map[string]bool{}
	activeEntries := map[string]bool{}
	seenEntries := map[string]bool{}
	for _, entry := range m.Rules {
		if seenEntries[entry.ID] {
			return nil, fmt.Errorf("duplicate rule entry %s", entry.ID)
		}
		seenEntries[entry.ID] = true
		if entry.Status != "active" {
			continue
		}
		def, err := r.Manifest.RuleDefinition(entry.Path)
		if err != nil {
			return nil, err
		}
		if def.ID != entry.ID {
			return nil, fmt.Errorf("active rule %s definition declares %s", entry.ID, def.ID)
		}
		var implementationRequirements []string
		testable := true
		switch def.Verification {
		case string(ClassSemantic):
			implementation, ok := r.Rules[entry.ID]
			if !ok {
				return nil, fmt.Errorf("active semantic rule %s has no implementation", entry.ID)
			}
			implementationRequirements = implementation.Requirements()
		case string(ClassBehavioral):
			implementation, ok := r.Behavioral[entry.ID]
			if !ok {
				return nil, fmt.Errorf("active behavioral rule %s has no implementation", entry.ID)
			}
			implementationRequirements = implementation.Requirements()
			testable = r.subjects().Has(implementation.Kind)
		default:
			return nil, fmt.Errorf("active rule %s declares unsupported verification %s", entry.ID, def.Verification)
		}
		activeEntries[entry.ID] = true
		if !activeProfiles[def.Profile][entry.ID] {
			return nil, fmt.Errorf("active rule %s is not listed by active profile %s", entry.ID, def.Profile)
		}
		definitionRequirements := append([]string{}, def.Requirements...)
		implementationRequirements = append([]string{}, implementationRequirements...)
		slices.Sort(definitionRequirements)
		slices.Sort(implementationRequirements)
		if !slices.Equal(definitionRequirements, implementationRequirements) {
			return nil, fmt.Errorf("active rule %s requirement citations differ between definition and implementation", entry.ID)
		}
		if !testable {
			continue
		}
		dirs[filepath.Join(r.Manifest.Dir, def.Fixtures.Valid)] = true
		dirs[filepath.Join(r.Manifest.Dir, def.Fixtures.Invalid)] = true
	}
	for profile, rules := range activeProfiles {
		for id := range rules {
			if !activeEntries[id] {
				return nil, fmt.Errorf("active profile %s lists non-active rule %s", profile, id)
			}
		}
	}
	keys := make([]string, 0, len(dirs))
	for d := range dirs {
		keys = append(keys, d)
	}
	sort.Strings(keys)
	var out []FixtureResult
	for _, d := range keys {
		res, err := r.RunFixture(d)
		if err != nil {
			return out, err
		}
		out = append(out, res)
	}
	return out, nil
}
