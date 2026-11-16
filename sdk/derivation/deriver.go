// Package derivation adopts documents of one document set into another, recording each copy's derivation
// (CHR-CONF-015).
package derivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/validate"
)

// notDerived are the kinds never copied: the target's own enterprise replaces the source's, and a record of what
// happened in the source set is never a record of the target's.
var notDerived = map[model.Kind]bool{
	model.KindEnterprise: true, model.KindActionRecord: true, model.KindEvidenceRecord: true, model.KindEvidenceBundle: true,
	model.KindExceptionRecord: true, model.KindEscalation: true, model.KindMediationDecision: true, model.KindApproval: true,
	model.KindProcessInstance: true, model.KindTaskInstance: true, model.KindBindingConformance: true,
	model.KindConformanceClaim: true,
}

// Deriver copies root documents of a source namespace, and the source documents they reference, into a target
// namespace.
type Deriver struct {
	meta *validate.SchemaMeta
}

func NewDeriver() (*Deriver, error) {
	meta, err := validate.NewSchemaMeta()
	if err != nil {
		return nil, err
	}
	return &Deriver{meta: meta}, nil
}

// pending is a source document to copy and the version label to read it at.
type pending struct {
	id, version string
}

// Derive returns, ordered by identifier, the copies to save: each root at its version and every source document it
// reaches through the schema's idRef and objectRef properties at its current version. A copy lives in the target
// namespace with the target enterprise, names its references to the source namespace in the target, drops the
// source's evidence and external identifiers, and records the source, its version, and its content digest. A target
// document already derived from the same version and content is reused, and one derived from another is reused or,
// with Upgrade, replaced; a target document of a copied identifier derived from nothing else is refused.
func (dv *Deriver) Derive(ctx context.Context, source corpus.VersionSource, target corpus.Source, roots []Root, opts Options) ([]*model.Document, error) {
	if dv == nil || dv.meta == nil || source == nil || target == nil {
		return nil, fmt.Errorf("%w: deriver, source, and target are required", ErrInvalidRequest)
	}
	if err := opts.check(roots); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var queue []pending
	for _, r := range roots {
		if !seen[r.ID] {
			seen[r.ID] = true
			queue = append(queue, pending{r.ID, r.Version})
		}
	}
	var out []*model.Document
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		key := corpus.DocumentKey{Namespace: opts.SourceNamespace, ID: next.id}
		s, err := source.FetchVersion(ctx, key, next.version)
		if err != nil {
			return nil, fmt.Errorf("source %s %q: %w", key, next.version, err)
		}
		c, refs, err := dv.copyOf(s, opts)
		if err != nil {
			return nil, err
		}
		for _, id := range refs {
			if !seen[id] {
				seen[id] = true
				queue = append(queue, pending{id: id})
			}
		}
		write, err := dv.replaces(ctx, target, c, opts.Upgrade)
		if err != nil {
			return nil, err
		}
		if write {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// copyOf builds the copy of a source document and lists the source identifiers it references.
func (dv *Deriver) copyOf(s *model.Document, opts Options) (*model.Document, []string, error) {
	if notDerived[s.Kind] {
		return nil, nil, fmt.Errorf("%w: %s %s", ErrNotDerivable, s.Kind, s.ID)
	}
	version, versioned := corpus.DefinitionVersion(s)
	if !versioned || version == "" {
		version = opts.SourceVersion
	}
	if version == "" {
		return nil, nil, fmt.Errorf("%w: %s %s", ErrNoSourceVersion, s.Kind, s.ID)
	}
	digest, err := corpus.ContentDigest(s)
	if err != nil {
		return nil, nil, err
	}
	cloned, err := dv.generic(s)
	if err != nil {
		return nil, nil, fmt.Errorf("copy %s: %w", s.ID, err)
	}
	value, ok := cloned.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("copy %s: document is not an object", s.ID)
	}
	value["namespace"] = opts.TargetNamespace
	delete(value, "externalIdentifiers")
	opts.dropSourceEvidence(value)
	if _, scoped := value["enterpriseId"]; scoped {
		if opts.Enterprise.ID == "" {
			return nil, nil, fmt.Errorf("%w: %s %s", ErrNoEnterprise, s.Kind, s.ID)
		}
		if value["enterpriseId"], err = dv.generic(opts.Enterprise); err != nil {
			return nil, nil, err
		}
	}
	refs := dv.relocate(value, opts)
	if value["derivedFrom"], err = dv.generic(model.Provenance{
		Source:  model.ObjectRef{Kind: s.Kind, Namespace: opts.SourceNamespace, ID: s.ID},
		Version: version, Digest: digest, Adoption: model.AdoptionAdopted,
	}); err != nil {
		return nil, nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, nil, fmt.Errorf("copy %s: %w", s.ID, err)
	}
	c, err := (corpus.Parser{}).Parse(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("copy %s: %w", s.ID, err)
	}
	return c, refs, nil
}

// relocate names each explicitly namespaced reference to the source namespace in the target namespace, where its copy
// lives, and returns the source identifiers referenced; the enterprise is the target's and no longer followed.
func (dv *Deriver) relocate(value map[string]any, opts Options) []string {
	inner := make(map[string]any, len(value))
	for k, x := range value {
		if k != "id" && k != "kind" && k != "enterpriseId" {
			inner[k] = x
		}
	}
	var refs []string
	follow := func(key corpus.DocumentKey, item any) {
		if key.Namespace != opts.SourceNamespace {
			return
		}
		refs = append(refs, key.ID)
		if n, ok := item.(map[string]any); ok && n["namespace"] != nil {
			n["namespace"] = opts.TargetNamespace
		}
	}
	dv.meta.WalkRefs(inner, func(_ string, ref model.Ref, item any) {
		follow(corpus.KeyOf(opts.SourceNamespace, ref), item)
	}, func(_ string, ref model.ObjectRef, item any) {
		if dv.meta.Kinds[string(ref.Kind)] && !ref.External {
			follow(corpus.ObjectKeyOf(opts.SourceNamespace, ref), item)
		}
	})
	return refs
}

// replaces decides whether a copy is written over what the target holds under its identifier.
func (dv *Deriver) replaces(ctx context.Context, target corpus.Source, c *model.Document, upgrade bool) (bool, error) {
	key := corpus.DocumentKey{Namespace: c.Namespace, ID: c.ID}
	existing, err := target.Fetch(ctx, key)
	if errors.Is(err, corpus.ErrNotFound) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("target %s: %w", key, err)
	}
	was, now := existing.DerivedFrom, c.DerivedFrom
	if existing.Kind != c.Kind || was == nil || was.Source.Kind != now.Source.Kind ||
		corpus.ObjectKeyOf(existing.Namespace, was.Source) != corpus.ObjectKeyOf(c.Namespace, now.Source) {
		return false, fmt.Errorf("%w: %s", ErrForeignDocument, key)
	}
	if (was.Version == now.Version && was.Digest == now.Digest) || !upgrade {
		return false, nil
	}
	if was.Adoption == model.AdoptionTailored {
		return false, fmt.Errorf("%w: %s", ErrTailored, key)
	}
	label, versioned := corpus.DefinitionVersion(c)
	stored, _ := corpus.DefinitionVersion(existing)
	if corpus.Published(existing) && (!versioned || label == stored) {
		return false, fmt.Errorf("%w: %s %q", ErrDerivationFixed, key, stored)
	}
	return true, nil
}

// generic returns a value as decoded JSON, numbers kept exact.
func (dv *Deriver) generic(v any) (any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
