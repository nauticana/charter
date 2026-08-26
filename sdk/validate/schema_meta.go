package validate

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/nauticana/charter/sdk/model"
)

// SchemaMeta derives the Charter kind set, every idRef- and objectRef-typed property name, and the document kinds each
// idRef property may name (its x-charter-ref-kinds annotation) from the embedded schemas.
type SchemaMeta struct {
	Kinds         map[string]bool
	IDRefKeys     map[string]bool
	ObjectRefKeys map[string]bool
	RefKinds      map[string][]model.Kind
}

func NewSchemaMeta() (*SchemaMeta, error) {
	m := &SchemaMeta{Kinds: map[string]bool{}, IDRefKeys: map[string]bool{}, ObjectRefKeys: map[string]bool{}, RefKinds: map[string][]model.Kind{}}
	var cat Catalog
	entries, err := cat.Entries()
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.Kind != "" {
			m.Kinds[e.Kind] = true
		}
		b, err := cat.Read(e)
		if err != nil {
			return nil, err
		}
		var v any
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, err
		}
		m.collect(v)
	}
	return m, nil
}

func (m *SchemaMeta) isIDRef(v any) bool {
	return schemaContainsRef(v, "/idRef")
}

func (m *SchemaMeta) isObjectRef(v any) bool {
	return schemaContainsRef(v, "/objectRef")
}

func schemaContainsRef(v any, suffix string) bool {
	sub, ok := v.(map[string]any)
	if !ok {
		return false
	}
	if r, ok := sub["$ref"].(string); ok && strings.HasSuffix(r, suffix) {
		return true
	}
	for _, key := range []string{"items", "allOf", "anyOf", "oneOf"} {
		child, found := sub[key]
		if !found {
			continue
		}
		switch child := child.(type) {
		case map[string]any:
			if schemaContainsRef(child, suffix) {
				return true
			}
		case []any:
			for _, item := range child {
				if schemaContainsRef(item, suffix) {
					return true
				}
			}
		}
	}
	return false
}

func (m *SchemaMeta) collect(v any) {
	switch n := v.(type) {
	case map[string]any:
		if props, ok := n["properties"].(map[string]any); ok {
			for k, p := range props {
				if m.isIDRef(p) {
					m.IDRefKeys[k] = true
					if annotated, ok := p.(map[string]any); ok {
						if kinds, ok := annotated["x-charter-ref-kinds"].([]any); ok {
							for _, kind := range kinds {
								if name, ok := kind.(string); ok && !slices.Contains(m.RefKinds[k], model.Kind(name)) {
									m.RefKinds[k] = append(m.RefKinds[k], model.Kind(name))
								}
							}
						}
					}
				}
				if m.isObjectRef(p) {
					m.ObjectRefKeys[k] = true
				}
			}
		}
		for _, x := range n {
			m.collect(x)
		}
	case []any:
		for _, x := range n {
			m.collect(x)
		}
	}
}
