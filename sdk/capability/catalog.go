// Package capability defines capability catalogs, version compatibility, and governed invocation: authority, approval,
// separation of duties, binding support, idempotency, outcome verification, and evidence around an abstract transport.
package capability

import (
	"context"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

type Catalog interface {
	Contract(ctx context.Context, ownerNamespace string, ref model.Ref) (model.CapabilityContract, error)
	Contracts(ctx context.Context) ([]model.CapabilityContract, error)
}

// BaseCatalog serves capability contracts from any corpus.Source.
type BaseCatalog struct {
	corpus.AbstractDocumentProvider
}

var _ Catalog = (*BaseCatalog)(nil)

func NewBaseCatalog(s corpus.Source) *BaseCatalog {
	return &BaseCatalog{corpus.AbstractDocumentProvider{Source: s}}
}

func (c *BaseCatalog) Contract(ctx context.Context, owner string, ref model.Ref) (model.CapabilityContract, error) {
	return corpus.ResolveAs[model.CapabilityContract](ctx, &c.AbstractDocumentProvider, owner, ref, model.KindCapabilityContract)
}

func (c *BaseCatalog) Contracts(ctx context.Context) ([]model.CapabilityContract, error) {
	return corpus.ListAs[model.CapabilityContract](ctx, &c.AbstractDocumentProvider, model.KindCapabilityContract)
}
