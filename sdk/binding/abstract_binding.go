package binding

import (
	"context"
	"errors"
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// VendorMapping is the downstream, vendor-specific part of a binding: request and response mapping, the call itself,
// and the translation of vendor errors into declared business errors (CHR-BIND-005, CHR-CAP-005).
type VendorMapping interface {
	MapRequest(ctx context.Context, req Request) (payload any, err error)
	Call(ctx context.Context, payload any) (result any, err error)
	MapResponse(ctx context.Context, result any) (Response, error)
	MapError(err error) (businessError string, ok bool)
}

// AbstractBinding realizes one CapabilityBinding document: it enforces declared feature support and fails closed
// around an abstract VendorMapping (CHR-BIND-006, CHR-CAP-007).
type AbstractBinding struct {
	Document model.CapabilityBinding
	Vendor   VendorMapping
}

var _ Executor = (*AbstractBinding)(nil)

func (b *AbstractBinding) Execute(ctx context.Context, req Request) (Response, error) {
	if b.Vendor == nil {
		return Response{}, fmt.Errorf("%w: binding %s has no vendor mapping", ErrNotExecuted, b.Document.ID)
	}
	if corpus.KeyOf(b.Document.Namespace, b.Document.CapabilityID) != corpus.KeyOf(b.Document.Namespace, req.Capability) {
		return Response{}, fmt.Errorf("%w: binding %s realizes %s, not %s", ErrNotExecuted, b.Document.ID, b.Document.CapabilityID.ID, req.Capability.ID)
	}
	if err := Features(b.Document.FeatureSupport).Require(req.RequiredFeatures...); err != nil {
		return Response{}, fmt.Errorf("%w: %v", ErrNotExecuted, err)
	}
	payload, err := b.Vendor.MapRequest(ctx, req)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %v", ErrNotExecuted, err)
	}
	result, err := b.Vendor.Call(ctx, payload)
	if err != nil {
		if businessError, ok := b.Vendor.MapError(err); ok {
			return Response{BusinessError: businessError}, nil
		}
		if errors.Is(err, ErrNotExecuted) {
			return Response{}, err
		}
		return Response{}, fmt.Errorf("%w: %v", ErrOutcomeUnknown, err)
	}
	resp, err := b.Vendor.MapResponse(ctx, result)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %v", ErrOutcomeUnknown, err)
	}
	return resp, nil
}

// Lossy lists the declared lossy mappings so callers can judge their effect on validation, decisions, and evidence (CHR-BIND-007).
func (b *AbstractBinding) Lossy() []model.LossyMapping { return b.Document.LossyMappings }
