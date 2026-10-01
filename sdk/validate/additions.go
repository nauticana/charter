package validate

import (
	"context"
	"errors"
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// Additions validates documents before they join or replace documents in a source: structure, the specification
// version of the embedded catalog, and references resolved against the source. Only the documents the written ones
// reference are fetched, so the cost follows the write rather than the size of the source. Findings about documents
// already stored are not reported, so an existing defect never blocks an unrelated write.
type Additions struct {
	structural *Structural
	meta       *SchemaMeta
	version    SpecVersion
	references []Validator
}

func NewAdditions() (*Additions, error) {
	structural, err := NewStructural()
	if err != nil {
		return nil, err
	}
	meta, err := NewSchemaMeta()
	if err != nil {
		return nil, err
	}
	catalog, err := (Catalog{}).Meta()
	if err != nil {
		return nil, err
	}
	rules, err := NewRuleSet()
	if err != nil {
		return nil, err
	}
	a := &Additions{structural: structural, meta: meta, version: SpecVersion{Version: catalog.SpecVersion}}
	for _, id := range []string{"CHR-RULE-CONF-001", "CHR-RULE-CONF-002", "CHR-RULE-ENT-005"} {
		rule, ok := rules[id]
		if !ok {
			return nil, fmt.Errorf("rule %s is not in the active rule set", id)
		}
		a.references = append(a.references, rule)
	}
	return a, nil
}

// Validate checks docs as they would stand in source, each replacing any stored document with the same key; a nil
// source holds no documents. An error is a failure to read the source, never a finding.
func (a *Additions) Validate(ctx context.Context, source corpus.Source, docs ...*model.Document) ([]Finding, error) {
	added := map[corpus.DocumentKey]bool{}
	var out []Finding
	written := corpus.New()
	for _, d := range docs {
		if d == nil {
			out = append(out, Finding{RuleID: "schema", Class: ClassStructural, Message: "nil document"})
			continue
		}
		key := corpus.DocumentKey{Namespace: d.Namespace, ID: d.ID}
		if added[key] {
			out = append(out, Finding{RuleID: "schema", Class: ClassStructural, Namespace: d.Namespace, DocumentID: d.ID, Message: "document appears twice in one write"})
			continue
		}
		added[key] = true
		out = append(out, a.structural.ValidateDocument(d)...)
		if err := written.Add(d); err != nil {
			return nil, err
		}
	}
	scope, err := a.withTargets(ctx, source, written)
	if err != nil {
		return nil, err
	}
	for _, v := range append([]Validator{a.version}, a.references...) {
		for _, f := range v.Validate(scope) {
			if added[corpus.DocumentKey{Namespace: f.Namespace, ID: f.DocumentID}] {
				out = append(out, f)
			}
		}
	}
	return out, nil
}

// withTargets adds to the written documents every stored document they reference; the reference rules judge a
// document only by what it points to, so nothing else in the source can change their findings.
func (a *Additions) withTargets(ctx context.Context, source corpus.Source, written *corpus.Corpus) (*corpus.Corpus, error) {
	scope := corpus.New()
	targets := map[corpus.DocumentKey]bool{}
	for _, d := range written.Documents() {
		if err := scope.Add(d); err != nil {
			return nil, err
		}
		walkRefs(a.meta, body(d), func(_ string, ref model.Ref) {
			targets[corpus.KeyOf(d.Namespace, ref)] = true
		}, func(_ string, ref model.ObjectRef) {
			if a.meta.Kinds[string(ref.Kind)] && !ref.External {
				targets[corpus.ObjectKeyOf(d.Namespace, ref)] = true
			}
		})
	}
	if source == nil {
		return scope, nil
	}
	for key := range targets {
		if _, inWrite := written.Get(key.Namespace, key.ID); inWrite {
			continue
		}
		d, err := source.Fetch(ctx, key)
		if errors.Is(err, corpus.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("reference %s: %w", key, err)
		}
		if (corpus.DocumentKey{Namespace: d.Namespace, ID: d.ID}) != key {
			return nil, fmt.Errorf("reference %s: source returned %s:%s", key, d.Namespace, d.ID)
		}
		if err := scope.Add(d); err != nil {
			return nil, err
		}
	}
	return scope, nil
}
