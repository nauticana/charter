package evidence

import (
	"context"
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// Verifier recomputes bundle integrity from the stored records (CHR-EVID-005).
type Verifier struct {
	Source   corpus.Source
	Digester Digester
}

func (v Verifier) Verify(ctx context.Context, ownerNamespace string, bundle model.Ref) error {
	d, err := v.Source.Fetch(ctx, corpus.KeyOf(ownerNamespace, bundle))
	if err != nil {
		return err
	}
	b, err := corpus.Decode[model.EvidenceBundle](d)
	if err != nil {
		return err
	}
	return v.check(ctx, b)
}

func (v Verifier) check(ctx context.Context, b model.EvidenceBundle) error {
	if b.Integrity.Method != v.Digester.Method() {
		return fmt.Errorf("%w: method %q is not %s", ErrIntegrity, b.Integrity.Method, v.Digester.Method())
	}
	value, err := v.chain(ctx, b)
	if err != nil {
		return err
	}
	if value != b.Integrity.Value {
		return fmt.Errorf("%w: bundle %s", ErrIntegrity, b.ID)
	}
	return nil
}

func (v Verifier) chain(ctx context.Context, b model.EvidenceBundle) (string, error) {
	digests := make([]string, 0, len(b.RecordIDs))
	for _, ref := range b.RecordIDs {
		d, err := v.Source.Fetch(ctx, corpus.KeyOf(b.Namespace, ref))
		if err != nil {
			return "", fmt.Errorf("%w: %s (%v)", ErrBundleRecord, ref.ID, err)
		}
		digests = append(digests, v.Digester.Digest(d.Raw))
	}
	return v.Digester.Chain(digests), nil
}
