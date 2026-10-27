package corpus

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nauticana/charter/sdk/model"
)

func TestCorpusKeysDocumentsByNamespaceAndID(t *testing.T) {
	c := New()
	for _, namespace := range []string{"one.example", "two.example"} {
		if err := c.Add(&model.Document{Namespace: namespace, ID: "SAME", Kind: model.KindEnterprise}); err != nil {
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

func TestDefinitionVersionReadsTheKindsVersionProperty(t *testing.T) {
	for _, tc := range []struct {
		raw       string
		version   string
		versioned bool
	}{
		{`{"id":"A","kind":"AgentDefinition","definitionVersion":"3"}`, "3", true},
		{`{"id":"P","kind":"BusinessProcess","processDefinitionVersion":"2"}`, "2", true},
		{`{"id":"T","kind":"Task","taskDefinitionVersion":"1"}`, "1", true},
		{`{"id":"C","kind":"CapabilityContract","contractVersion":"1.4"}`, "1.4", true},
		{`{"id":"B","kind":"AuthorityBinding","bindingVersion":"7"}`, "7", true},
		{`{"id":"S","kind":"SystemProfile","systemVersion":"2602"}`, "", false},
		{`{"id":"T","kind":"Task"}`, "", true},
	} {
		d, err := (Parser{}).Parse([]byte(tc.raw))
		if err != nil {
			t.Fatal(err)
		}
		if version, versioned := DefinitionVersion(d); version != tc.version || versioned != tc.versioned {
			t.Errorf("%s = %q %v, want %q %v", tc.raw, version, versioned, tc.version, tc.versioned)
		}
	}
}

func TestUnionServesTheFirstSourceHoldingAKey(t *testing.T) {
	one, two := New(), New()
	for i, c := range []*Corpus{one, two} {
		for _, id := range []string{"SHARED", []string{"ONLY-ONE", "ONLY-TWO"}[i]} {
			if err := c.Add(&model.Document{Namespace: "n", ID: id, Kind: model.KindEnterprise, Name: id + string(rune('1'+i))}); err != nil {
				t.Fatal(err)
			}
		}
	}
	ctx, u := context.Background(), Union{one, two}
	if d, err := u.Fetch(ctx, DocumentKey{Namespace: "n", ID: "SHARED"}); err != nil || d.Name != "SHARED1" {
		t.Errorf("shared key: %+v %v", d, err)
	}
	if d, err := u.Fetch(ctx, DocumentKey{Namespace: "n", ID: "ONLY-TWO"}); err != nil || d.ID != "ONLY-TWO" {
		t.Errorf("second source: %+v %v", d, err)
	}
	if _, err := u.Fetch(ctx, DocumentKey{Namespace: "n", ID: "NONE"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing key: %v", err)
	}
	if docs, err := u.List(ctx, model.KindEnterprise); err != nil || len(docs) != 4 {
		t.Errorf("list: %d %v", len(docs), err)
	}
}

func TestDerivationsNameTheirSource(t *testing.T) {
	c := New()
	source := model.ObjectRef{Kind: model.KindResponsibility, ID: "RESP-BLUEPRINT", Namespace: "blueprint.example"}
	for id, from := range map[string]*model.Provenance{
		"ADOPTED":  {Source: source, Version: "1", Digest: "sha256:" + strings.Repeat("0", 64), Adoption: model.AdoptionAdopted},
		"TAILORED": {Source: source, Version: "2", Digest: "sha256:" + strings.Repeat("1", 64), Adoption: model.AdoptionTailored},
		"OTHER":    {Source: model.ObjectRef{Kind: model.KindRole, ID: "RESP-BLUEPRINT", Namespace: "blueprint.example"}, Version: "1"},
		"OWN":      nil,
	} {
		d := &model.Document{Namespace: "customer.example", ID: id, Kind: model.KindResponsibility, DerivedFrom: from}
		if err := c.Add(d); err != nil {
			t.Fatal(err)
		}
	}
	got := c.Derivations("customer.example", source)
	if len(got) != 2 || got[0].ID != "ADOPTED" || got[1].ID != "TAILORED" {
		t.Fatalf("derivations = %v", got)
	}
	if got := c.Derivations("customer.example", model.ObjectRef{Kind: model.KindResponsibility, ID: "RESP-BLUEPRINT"}); len(got) != 0 {
		t.Fatalf("a source in another namespace matched a local one: %v", got)
	}
}
