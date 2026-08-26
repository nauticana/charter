package schema

import (
	"encoding/json"
	"io/fs"
	"testing"
)

func TestCatalogEntriesAreEmbeddedWithMatchingIDs(t *testing.T) {
	b, err := fs.ReadFile(FS, "catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	var cat struct {
		Entries []struct {
			Kind string `json:"kind"`
			ID   string `json:"id"`
			Path string `json:"path"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(b, &cat); err != nil {
		t.Fatal(err)
	}
	if len(cat.Entries) != 44 {
		t.Errorf("catalog lists %d entries, want common, three conformance formats, and 40 kinds", len(cat.Entries))
	}
	kinds := map[string]bool{}
	for _, e := range cat.Entries {
		raw, err := fs.ReadFile(FS, e.Path)
		if err != nil {
			t.Errorf("%s: %v", e.Path, err)
			continue
		}
		var s struct {
			ID    string `json:"$id"`
			Title string `json:"title"`
			Props struct {
				Kind struct {
					Const string `json:"const"`
				} `json:"kind"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Errorf("%s: %v", e.Path, err)
			continue
		}
		if s.ID != e.ID {
			t.Errorf("%s: $id %s differs from catalog id %s", e.Path, s.ID, e.ID)
		}
		if e.Kind != "" && (s.Props.Kind.Const != e.Kind || kinds[e.Kind]) {
			t.Errorf("%s: kind const %q for catalog kind %q (duplicate=%v)", e.Path, s.Props.Kind.Const, e.Kind, kinds[e.Kind])
		}
		kinds[e.Kind] = true
	}
}
