package validate

import (
	"fmt"
	"slices"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// ReferencesResolveRule checks that every idRef resolves and every Charter-kind objectRef resolves to a document of the declared kind.
type ReferencesResolveRule struct {
	AbstractRule
	Meta *SchemaMeta
}

// ReferenceKindsRule checks that every resolved idRef names one of the kinds its property declares in x-charter-ref-kinds.
type ReferenceKindsRule struct {
	AbstractRule
	Meta *SchemaMeta
}

// EnterpriseBoundaryRule checks that every resolved reference from an enterprise-scoped document stays within that enterprise.
type EnterpriseBoundaryRule struct {
	AbstractRule
	Meta *SchemaMeta
}

var (
	_ Rule = ReferencesResolveRule{}
	_ Rule = ReferenceKindsRule{}
	_ Rule = EnterpriseBoundaryRule{}
)

func (r ReferencesResolveRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, d := range c.Documents() {
		walkRefs(r.Meta, body(d), func(key string, ref model.Ref) {
			if _, found := c.Resolve(d.Namespace, ref); !found {
				out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("%s %s not found", key, ref.ID)))
			}
		}, func(_ string, ref model.ObjectRef) {
			out = append(out, r.validateObjectRef(c, d.Namespace, d.ID, ref)...)
		})
	}
	return out
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

func (r ReferenceKindsRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, d := range c.Documents() {
		walkRefs(r.Meta, body(d), func(key string, ref model.Ref) {
			target, found := c.Resolve(d.Namespace, ref)
			if !found {
				return
			}
			kinds, known := r.Meta.RefKinds[key]
			if !known {
				out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("%s declares no x-charter-ref-kinds", key)))
				return
			}
			if !slices.Contains(kinds, target.Kind) {
				out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("%s %s is %s, not %v", key, ref.ID, target.Kind, kinds)))
			}
		}, nil)
	}
	return out
}

func (r EnterpriseBoundaryRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, d := range c.Documents() {
		if d.EnterpriseID == nil {
			continue
		}
		owner := corpus.KeyOf(d.Namespace, *d.EnterpriseID)
		check := func(key, id string, target *model.Document) {
			if target.EnterpriseID != nil && corpus.KeyOf(target.Namespace, *target.EnterpriseID) != owner {
				out = append(out, r.finding(d.Namespace, d.ID, fmt.Sprintf("%s %s belongs to enterprise %s, not %s", key, id, target.EnterpriseID.ID, owner.ID)))
			}
		}
		walkRefs(r.Meta, body(d), func(key string, ref model.Ref) {
			if target, found := c.Resolve(d.Namespace, ref); found && key != "enterpriseId" {
				check(key, ref.ID, target)
			}
		}, func(key string, ref model.ObjectRef) {
			if target, found := c.ResolveObject(d.Namespace, ref); found && !ref.External {
				check(key, ref.ID, target)
			}
		})
	}
	return out
}

// body is the document value without its own id and kind, which are not references.
func body(d *model.Document) map[string]any {
	value, _ := d.Value.(map[string]any)
	inner := make(map[string]any, len(value))
	for k, x := range value {
		if k != "id" && k != "kind" {
			inner[k] = x
		}
	}
	return inner
}

// walkRefs visits every idRef and objectRef property below v, skipping namespaced extensions.
func walkRefs(meta *SchemaMeta, v any, onID func(key string, ref model.Ref), onObject func(key string, ref model.ObjectRef)) {
	switch n := v.(type) {
	case map[string]any:
		for k, x := range n {
			if k == "extensions" {
				continue
			}
			items := []any{x}
			if list, ok := x.([]any); ok {
				items = list
			}
			if meta.ObjectRefKeys[k] && onObject != nil {
				for _, item := range items {
					if ref, ok := objectRef(item); ok {
						onObject(k, ref)
					}
				}
			}
			if meta.IDRefKeys[k] && onID != nil {
				for _, item := range items {
					if ref, ok := idRef(item); ok {
						onID(k, ref)
					}
				}
			}
			walkRefs(meta, x, onID, onObject)
		}
	case []any:
		for _, x := range n {
			walkRefs(meta, x, onID, onObject)
		}
	}
}

func idRef(v any) (model.Ref, bool) {
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
