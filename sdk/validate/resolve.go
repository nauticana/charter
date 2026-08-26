package validate

import (
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// resolveAs returns the typed target of a reference when it resolves to the expected kind; dangling and
// wrongly-typed references are the reference rules' findings, so other rules skip them.
func resolveAs[T any](c *corpus.Corpus, ownerNamespace string, ref model.Ref, kind model.Kind) (T, bool) {
	var zero T
	d, ok := c.Resolve(ownerNamespace, ref)
	if !ok || d.Kind != kind {
		return zero, false
	}
	t, err := corpus.Decode[T](d)
	return t, err == nil
}

func allOf[T any](c *corpus.Corpus, kind model.Kind) []T {
	var out []T
	for _, d := range c.OfKind(kind) {
		if t, err := corpus.Decode[T](d); err == nil {
			out = append(out, t)
		}
	}
	return out
}
