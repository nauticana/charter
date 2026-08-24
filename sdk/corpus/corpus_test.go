package corpus

import (
	"testing"

	"github.com/nauticana/charter/sdk/model"
)

func TestCorpusKeysDocumentsByNamespaceAndID(t *testing.T) {
	c := New()
	for _, namespace := range []string{"one.example", "two.example"} {
		if err := c.Add(&model.Document{Envelope: model.Envelope{Namespace: namespace, ID: "SAME", Kind: model.KindEnterprise}}); err != nil {
			t.Fatal(err)
		}
	}
	if c.Len() != 2 {
		t.Fatalf("got %d documents, want 2", c.Len())
	}
	if d, ok := c.Resolve("one.example", model.Ref{ID: "SAME"}); !ok || d.Namespace != "one.example" {
		t.Fatal("same-namespace reference resolved incorrectly")
	}
	if d, ok := c.Resolve("one.example", model.Ref{Namespace: "two.example", ID: "SAME"}); !ok || d.Namespace != "two.example" {
		t.Fatal("cross-namespace reference resolved incorrectly")
	}
}
