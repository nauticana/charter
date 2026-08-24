package validate

import (
	"bytes"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	FormatVersion int `yaml:"format_version"`
	Specification struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"specification"`
	Profiles []struct {
		ID     string   `yaml:"id"`
		Status string   `yaml:"status"`
		Rules  []string `yaml:"rules"`
	} `yaml:"profiles"`
	Rules []struct {
		ID     string `yaml:"id"`
		Status string `yaml:"status"`
		Path   string `yaml:"path"`
	} `yaml:"rules"`
}

type RuleDefinition struct {
	ID           string   `yaml:"id"`
	Title        string   `yaml:"title"`
	Requirements []string `yaml:"requirements"`
	Profile      string   `yaml:"profile"`
	Verification string   `yaml:"verification"`
	AppliesTo    []string `yaml:"applies_to"`
	Check        string   `yaml:"check"`
	Fixtures     struct {
		Valid   string `yaml:"valid"`
		Invalid string `yaml:"invalid"`
	} `yaml:"fixtures"`
}

type FixtureDescriptor struct {
	Rules                []string `yaml:"rules"`
	AlsoFails            []string `yaml:"also_fails"`
	SpecificationVersion string   `yaml:"specification_version"`
	Expected             string   `yaml:"expected"`
	Reason               string   `yaml:"reason"`
	Documents            string   `yaml:"documents"`
}

// ManifestReader reads the conformance manifest, rule definitions, and fixture descriptors below Dir.
type ManifestReader struct {
	Dir string
}

func (m ManifestReader) read(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	return dec.Decode(v)
}

func (m ManifestReader) Manifest() (*Manifest, error) {
	var out Manifest
	return &out, m.read(filepath.Join(m.Dir, "manifest.yaml"), &out)
}

func (m ManifestReader) RuleDefinition(relPath string) (*RuleDefinition, error) {
	var out RuleDefinition
	return &out, m.read(filepath.Join(m.Dir, relPath), &out)
}

func (m ManifestReader) Fixture(dir string) (*FixtureDescriptor, error) {
	var out FixtureDescriptor
	return &out, m.read(filepath.Join(dir, "fixture.yaml"), &out)
}
