package validate

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type compiledSchema struct {
	schema       *jsonschema.Schema
	requirements []string
}

// Structural validates documents against the embedded schema for their kind.
type Structural struct {
	byKind map[model.Kind]compiledSchema
}

var _ Validator = (*Structural)(nil)

func NewStructural() (*Structural, error) {
	var cat Catalog
	entries, err := cat.Entries()
	if err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	reqs := map[string][]string{}
	for _, e := range entries {
		b, err := cat.Read(e)
		if err != nil {
			return nil, err
		}
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Path, err)
		}
		if err := c.AddResource(e.ID, doc); err != nil {
			return nil, err
		}
		var meta struct {
			Requirements []string `json:"x-charter-requirements"`
		}
		_ = json.Unmarshal(b, &meta)
		reqs[e.ID] = meta.Requirements
	}
	s := &Structural{byKind: map[model.Kind]compiledSchema{}}
	for _, e := range entries {
		if e.Kind == "" {
			continue
		}
		sch, err := c.Compile(e.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Path, err)
		}
		s.byKind[model.Kind(e.Kind)] = compiledSchema{schema: sch, requirements: reqs[e.ID]}
	}
	return s, nil
}

func (s *Structural) ValidateDocument(d *model.Document) []Finding {
	c, ok := s.byKind[d.Kind]
	if !ok {
		return []Finding{{RuleID: "schema", Class: ClassStructural, Namespace: d.Namespace, DocumentID: d.ID, Message: fmt.Sprintf("unknown kind %s", d.Kind)}}
	}
	err := c.schema.Validate(d.Value)
	if err == nil {
		return nil
	}
	ve, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return []Finding{{RuleID: "schema", Class: ClassStructural, Namespace: d.Namespace, DocumentID: d.ID, Message: err.Error()}}
	}
	var out []Finding
	for _, u := range ve.BasicOutput().Errors {
		if u.Error == nil {
			continue
		}
		out = append(out, Finding{RuleID: "schema:" + string(d.Kind), Requirements: c.requirements, Class: ClassStructural,
			Namespace: d.Namespace, DocumentID: d.ID, Path: u.InstanceLocation, Message: u.Error.String()})
	}
	return out
}

func (s *Structural) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, d := range c.Documents() {
		out = append(out, s.ValidateDocument(d)...)
	}
	return out
}
