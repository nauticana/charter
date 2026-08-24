package validate

import (
	"encoding/json"
	"strings"
)

// SchemaMeta derives the Charter kind set and every idRef-typed property name from the embedded schemas.
type SchemaMeta struct {
	Kinds         map[string]bool
	IDRefKeys     map[string]bool
	ObjectRefKeys map[string]bool
}

func NewSchemaMeta() (*SchemaMeta, error) {
	m := &SchemaMeta{Kinds: map[string]bool{}, IDRefKeys: map[string]bool{}, ObjectRefKeys: map[string]bool{}}
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
