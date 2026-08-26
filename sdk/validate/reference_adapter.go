package validate

import (
	"context"
	"fmt"
	"slices"

	"github.com/nauticana/charter/sdk/binding"
	"github.com/nauticana/charter/sdk/capability"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// ReferenceAdapter realizes a capability binding with the SDK's AbstractBinding over a neutral vendor protocol: the
// vendor answers with a result whose status names a declared outcome (plus an optional reference), or with an error
// whose text names a declared business error. It is the adapter the manifest runs against unless one is supplied.
type ReferenceAdapter struct{}

var _ AdapterSubject = ReferenceAdapter{}

func (ReferenceAdapter) Realize(ctx context.Context, documents corpus.Source, ownerNamespace string, ref model.Ref, vendor VendorEndpoint) (binding.Executor, error) {
	if vendor == nil {
		return nil, fmt.Errorf("binding %s has no vendor endpoint", ref.ID)
	}
	doc, err := binding.NewBaseProvider(documents).CapabilityBinding(ctx, ownerNamespace, ref)
	if err != nil {
		return nil, err
	}
	contract, err := capability.NewBaseCatalog(documents).Contract(ctx, doc.Namespace, doc.CapabilityID)
	if err != nil {
		return nil, err
	}
	return &binding.AbstractBinding{Document: doc, Vendor: referenceVendorMapping{contract: contract, vendor: vendor}}, nil
}

type referenceVendorMapping struct {
	contract model.CapabilityContract
	vendor   VendorEndpoint
}

var _ binding.VendorMapping = referenceVendorMapping{}

func (m referenceVendorMapping) MapRequest(_ context.Context, req binding.Request) (any, error) {
	return map[string]any{"capability": req.Capability.ID, "inputs": req.Inputs, "idempotency_key": req.IdempotencyKey}, nil
}

func (m referenceVendorMapping) Call(ctx context.Context, payload any) (any, error) {
	return m.vendor.Call(ctx, payload)
}

func (m referenceVendorMapping) MapResponse(_ context.Context, result any) (binding.Response, error) {
	fields, _ := result.(map[string]any)
	status, _ := fields["status"].(string)
	if !slices.Contains(m.contract.Outcomes, status) {
		return binding.Response{}, fmt.Errorf("vendor status %q names no declared outcome of %s", status, m.contract.ID)
	}
	reference, _ := fields["reference"].(string)
	return binding.Response{Outcome: status, ExternalReference: reference}, nil
}

func (m referenceVendorMapping) MapError(err error) (string, bool) {
	if slices.Contains(m.contract.BusinessErrors, err.Error()) {
		return err.Error(), true
	}
	return "", false
}
