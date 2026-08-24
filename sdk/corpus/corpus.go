// Package corpus loads and indexes sets of Charter documents.
package corpus

import (
	"fmt"
	"sort"

	"github.com/nauticana/charter/sdk/model"
)

// DocumentKey is the stable address of a Charter document.
type DocumentKey struct {
	Namespace string
	ID        string
}

// Corpus is a set of documents addressed by namespace and id, the unit over which semantic rules run.
type Corpus struct {
	docs map[DocumentKey]*model.Document
}

func New() *Corpus { return &Corpus{docs: map[DocumentKey]*model.Document{}} }

func (c *Corpus) Add(d *model.Document) error {
	k := DocumentKey{Namespace: d.Namespace, ID: d.ID}
	if _, dup := c.docs[k]; dup {
		return fmt.Errorf("duplicate document %s:%s", d.Namespace, d.ID)
	}
	c.docs[k] = d
	return nil
}

func (c *Corpus) Get(namespace, id string) (*model.Document, bool) {
	d, ok := c.docs[DocumentKey{Namespace: namespace, ID: id}]
	return d, ok
}

// Resolve resolves an idRef relative to its owning document's namespace.
func (c *Corpus) Resolve(ownerNamespace string, ref model.Ref) (*model.Document, bool) {
	namespace := ref.Namespace
	if namespace == "" {
		namespace = ownerNamespace
	}
	return c.Get(namespace, ref.ID)
}

// ResolveObject resolves an objectRef relative to its owning document's namespace.
func (c *Corpus) ResolveObject(ownerNamespace string, ref model.ObjectRef) (*model.Document, bool) {
	return c.Resolve(ownerNamespace, model.Ref{Namespace: ref.Namespace, ID: ref.ID})
}

// KindOfRef returns the kind of an idRef target, or empty when it is absent.
func (c *Corpus) KindOfRef(ownerNamespace string, ref model.Ref) model.Kind {
	if d, ok := c.Resolve(ownerNamespace, ref); ok {
		return d.Kind
	}
	return ""
}

// Documents returns all documents ordered by id.
func (c *Corpus) Documents() []*model.Document {
	out := make([]*model.Document, 0, len(c.docs))
	for _, d := range c.docs {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace == out[j].Namespace {
			return out[i].ID < out[j].ID
		}
		return out[i].Namespace < out[j].Namespace
	})
	return out
}

func (c *Corpus) OfKind(k model.Kind) []*model.Document {
	var out []*model.Document
	for _, d := range c.Documents() {
		if d.Kind == k {
			out = append(out, d)
		}
	}
	return out
}

func (c *Corpus) Len() int { return len(c.docs) }
