package validate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"

	"github.com/nauticana/charter/schema"
)

const (
	formatManifest = "conformance-manifest"
	formatRule     = "conformance-rule-definition"
	formatFixture  = "conformance-fixture-descriptor"
)

// formats compiles the embedded conformance format schemas once.
var formats struct {
	once    sync.Once
	schemas map[string]*jsonschema.Schema
	err     error
}

func formatSchema(name string) (*jsonschema.Schema, error) {
	formats.once.Do(func() {
		formats.schemas = map[string]*jsonschema.Schema{}
		var cat Catalog
		entries, err := cat.Entries()
		if err != nil {
			formats.err = err
			return
		}
		compiler := jsonschema.NewCompiler()
		compiler.AssertFormat()
		var formatEntries []CatalogEntry
		for _, e := range entries {
			b, err := fs.ReadFile(schema.FS, e.Path)
			if err != nil {
				formats.err = err
				return
			}
			doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
			if err != nil {
				formats.err = fmt.Errorf("%s: %w", e.Path, err)
				return
			}
			if err := compiler.AddResource(e.ID, doc); err != nil {
				formats.err = err
				return
			}
			if e.Kind == "" && strings.HasPrefix(e.Name, "conformance-") {
				formatEntries = append(formatEntries, e)
			}
		}
		for _, e := range formatEntries {
			s, err := compiler.Compile(e.ID)
			if err != nil {
				formats.err = fmt.Errorf("%s: %w", e.Path, err)
				return
			}
			formats.schemas[e.Name] = s
		}
	})
	if formats.err != nil {
		return nil, formats.err
	}
	s, ok := formats.schemas[name]
	if !ok {
		return nil, fmt.Errorf("no embedded schema for format %s", name)
	}
	return s, nil
}

// validateFormat checks YAML bytes against a conformance format schema (CHR-CONF-003).
func validateFormat(name string, b []byte) error {
	s, err := formatSchema(name)
	if err != nil {
		return err
	}
	var value any
	if err := yaml.Unmarshal(b, &value); err != nil {
		return err
	}
	js, err := json.Marshal(value)
	if err != nil {
		return err
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(js))
	if err != nil {
		return err
	}
	err = s.Validate(instance)
	if err == nil {
		return nil
	}
	ve, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return err
	}
	var problems []string
	for _, u := range ve.BasicOutput().Errors {
		if u.Error != nil {
			problems = append(problems, u.InstanceLocation+": "+u.Error.String())
		}
	}
	return fmt.Errorf("%s format: %s", name, strings.Join(problems, "; "))
}
