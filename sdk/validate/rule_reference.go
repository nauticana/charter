package validate

import (
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// ReferencesResolveRule checks that every idRef and Charter-kind objectRef resolves to a document of the declared kind.
type ReferencesResolveRule struct {
	AbstractRule
	Meta *SchemaMeta
}

var _ Rule = ReferencesResolveRule{}

func (r ReferencesResolveRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, d := range c.Documents() {
		body, _ := d.Value.(map[string]any)
		inner := make(map[string]any, len(body))
		for k, x := range body {
			if k != "id" && k != "kind" {
				inner[k] = x
			}
		}
		out = append(out, r.walk(c, inner, d.Namespace, d.ID)...)
	}
	return out
}

func (r ReferencesResolveRule) ref(v any) (model.Ref, bool) {
	switch ref := v.(type) {
	case string:
		return model.Ref{ID: ref}, true
	case map[string]any:
		id, ok := ref["id"].(string)
		namespace, _ := ref["namespace"].(string)
		return model.Ref{Namespace: namespace, ID: id}, ok
	}
	return model.Ref{}, false
}

func (r ReferencesResolveRule) walk(c *corpus.Corpus, v any, ownerNamespace, ownerID string) []Finding {
	var out []Finding
	switch n := v.(type) {
	case map[string]any:
		for k, x := range n {
			if k == "extensions" {
				continue
			}
			if r.Meta.ObjectRefKeys[k] {
				items := []any{x}
				if list, ok := x.([]any); ok {
					items = list
				}
				for _, item := range items {
					if ref, ok := objectRef(item); ok {
						out = append(out, r.validateObjectRef(c, ownerNamespace, ownerID, ref)...)
					}
				}
			}
			if r.Meta.IDRefKeys[k] {
				items := []any{x}
				if list, ok := x.([]any); ok {
					items = list
				}
				for _, it := range items {
					if ref, ok := r.ref(it); ok {
						if _, found := c.Resolve(ownerNamespace, ref); !found {
							out = append(out, r.finding(ownerNamespace, ownerID, fmt.Sprintf("%s %s not found", k, ref.ID)))
						}
					}
				}
			}
			out = append(out, r.walk(c, x, ownerNamespace, ownerID)...)
		}
	case []any:
		for _, x := range n {
			out = append(out, r.walk(c, x, ownerNamespace, ownerID)...)
		}
	}
	return out
}

func objectRef(v any) (model.ObjectRef, bool) {
	n, ok := v.(map[string]any)
	if !ok {
		return model.ObjectRef{}, false
	}
	kind, kindOK := n["kind"].(string)
	id, idOK := n["id"].(string)
	namespace, _ := n["namespace"].(string)
	external, _ := n["external"].(bool)
	return model.ObjectRef{Kind: model.Kind(kind), Namespace: namespace, ID: id, External: external}, kindOK && idOK
}

func (r ReferencesResolveRule) validateObjectRef(c *corpus.Corpus, ownerNamespace, ownerID string, ref model.ObjectRef) []Finding {
	if !r.Meta.Kinds[string(ref.Kind)] {
		if ref.External {
			return nil
		}
		return []Finding{r.finding(ownerNamespace, ownerID, fmt.Sprintf("%s is not a Charter kind and is not declared external", ref.Kind))}
	}
	target, found := c.ResolveObject(ownerNamespace, ref)
	if !found {
		return []Finding{r.finding(ownerNamespace, ownerID, fmt.Sprintf("%s %s not found", ref.Kind, ref.ID))}
	}
	if target.Kind != ref.Kind {
		return []Finding{r.finding(ownerNamespace, ownerID, fmt.Sprintf("%s is %s, not %s", ref.ID, target.Kind, ref.Kind))}
	}
	return nil
}
