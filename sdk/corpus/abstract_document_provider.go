package corpus

import (
	"context"
	"fmt"
	"time"

	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/temporal"
)

// AbstractDocumentProvider adds kind validation, reference resolution, and effective-time queries to any Source;
// domain providers embed it and expose typed methods through ResolveAs, ListAs, and EffectiveAs.
type AbstractDocumentProvider struct {
	Source Source
}

func (p *AbstractDocumentProvider) Resolve(ctx context.Context, ownerNamespace string, ref model.Ref, kind model.Kind) (*model.Document, error) {
	if p.Source == nil {
		return nil, ErrNoSource
	}
	if ref.ID == "" {
		return nil, fmt.Errorf("%w: empty reference", ErrNotFound)
	}
	d, err := p.Source.Fetch(ctx, KeyOf(ownerNamespace, ref))
	if err != nil {
		return nil, err
	}
	if kind != "" && d.Kind != kind {
		return nil, fmt.Errorf("%w: %s is %s, not %s", ErrKindMismatch, d.ID, d.Kind, kind)
	}
	return d, nil
}

// ResolveObject resolves an objectRef to a document of the referenced kind; declared-external references never resolve.
func (p *AbstractDocumentProvider) ResolveObject(ctx context.Context, ownerNamespace string, ref model.ObjectRef) (*model.Document, error) {
	if ref.External {
		return nil, fmt.Errorf("%w: %s %s", ErrExternal, ref.Kind, ref.ID)
	}
	return p.Resolve(ctx, ownerNamespace, model.Ref{Namespace: ref.Namespace, ID: ref.ID}, ref.Kind)
}

func (p *AbstractDocumentProvider) List(ctx context.Context, kind model.Kind) ([]*model.Document, error) {
	if p.Source == nil {
		return nil, ErrNoSource
	}
	return p.Source.List(ctx, kind)
}

// EffectiveAt lists the documents of a kind whose validity contains at; an absent validity is open-ended.
func (p *AbstractDocumentProvider) EffectiveAt(ctx context.Context, kind model.Kind, at time.Time) ([]*model.Document, error) {
	docs, err := p.List(ctx, kind)
	if err != nil {
		return nil, err
	}
	var out []*model.Document
	for _, d := range docs {
		if temporal.EffectiveAt(d.Validity, at) {
			out = append(out, d)
		}
	}
	return out, nil
}

func ResolveAs[T any](ctx context.Context, p *AbstractDocumentProvider, ownerNamespace string, ref model.Ref, kind model.Kind) (T, error) {
	d, err := p.Resolve(ctx, ownerNamespace, ref, kind)
	if err != nil {
		var zero T
		return zero, err
	}
	return Decode[T](d)
}

func ResolveObjectAs[T any](ctx context.Context, p *AbstractDocumentProvider, ownerNamespace string, ref model.ObjectRef) (T, error) {
	d, err := p.ResolveObject(ctx, ownerNamespace, ref)
	if err != nil {
		var zero T
		return zero, err
	}
	return Decode[T](d)
}

func ListAs[T any](ctx context.Context, p *AbstractDocumentProvider, kind model.Kind) ([]T, error) {
	docs, err := p.List(ctx, kind)
	if err != nil {
		return nil, err
	}
	return decodeAll[T](docs)
}

func EffectiveAs[T any](ctx context.Context, p *AbstractDocumentProvider, kind model.Kind, at time.Time) ([]T, error) {
	docs, err := p.EffectiveAt(ctx, kind, at)
	if err != nil {
		return nil, err
	}
	return decodeAll[T](docs)
}

func decodeAll[T any](docs []*model.Document) ([]T, error) {
	out := make([]T, 0, len(docs))
	for _, d := range docs {
		t, err := Decode[T](d)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", d.ID, err)
		}
		out = append(out, t)
	}
	return out, nil
}
