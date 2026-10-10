package corpus

import (
	"context"
	"errors"
	"fmt"

	"github.com/nauticana/charter/sdk/model"
)

// Source is the abstract loading contract behind every document provider; Corpus is the in-memory implementation.
type Source interface {
	Fetch(ctx context.Context, key DocumentKey) (*model.Document, error)
	List(ctx context.Context, kind model.Kind) ([]*model.Document, error)
}

var _ Source = (*Corpus)(nil)

func (c *Corpus) Fetch(_ context.Context, key DocumentKey) (*model.Document, error) {
	d, ok := c.docs[key]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, key)
	}
	return d, nil
}

func (c *Corpus) List(_ context.Context, kind model.Kind) ([]*model.Document, error) {
	return c.OfKind(kind), nil
}

func (k DocumentKey) String() string { return k.Namespace + ":" + k.ID }

// Ref returns the explicitly namespaced idRef of the key.
func (k DocumentKey) Ref() model.Ref { return model.Ref{Namespace: k.Namespace, ID: k.ID} }

// KeyOf resolves an idRef relative to its owning document's namespace.
func KeyOf(ownerNamespace string, ref model.Ref) DocumentKey {
	if ref.Namespace != "" {
		return DocumentKey{Namespace: ref.Namespace, ID: ref.ID}
	}
	return DocumentKey{Namespace: ownerNamespace, ID: ref.ID}
}

func ObjectKeyOf(ownerNamespace string, ref model.ObjectRef) DocumentKey {
	return KeyOf(ownerNamespace, model.Ref{Namespace: ref.Namespace, ID: ref.ID})
}

// Union reads several sources as one, such as definitions and the evidence they cite; the first source holding a key serves it.
type Union []Source

var _ Source = Union(nil)

func (u Union) Fetch(ctx context.Context, key DocumentKey) (*model.Document, error) {
	for _, s := range u {
		d, err := s.Fetch(ctx, key)
		if !errors.Is(err, ErrNotFound) {
			return d, err
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNotFound, key)
}

func (u Union) List(ctx context.Context, kind model.Kind) ([]*model.Document, error) {
	var out []*model.Document
	for _, s := range u {
		docs, err := s.List(ctx, kind)
		if err != nil {
			return nil, err
		}
		out = append(out, docs...)
	}
	return out, nil
}
