package validate

import (
	"encoding/json"
	"io/fs"

	"github.com/nauticana/charter/schema"
)

type CatalogEntry struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Path string `json:"path"`
}

// Catalog reads the embedded schema catalog.
type Catalog struct{}

func (Catalog) Entries() ([]CatalogEntry, error) {
	b, err := fs.ReadFile(schema.FS, "catalog.json")
	if err != nil {
		return nil, err
	}
	var cat struct {
		Entries []CatalogEntry `json:"entries"`
	}
	return cat.Entries, json.Unmarshal(b, &cat)
}

func (Catalog) Read(e CatalogEntry) ([]byte, error) { return fs.ReadFile(schema.FS, e.Path) }
