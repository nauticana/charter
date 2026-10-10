package validate

import (
	"encoding/json"
	"fmt"
	"io/fs"

	"github.com/nauticana/charter/schema"
)

type CatalogEntry struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
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

// SpecVersion is the specification release the embedded schemas define, the one place the release is recorded.
func (Catalog) SpecVersion() (string, error) {
	b, err := fs.ReadFile(schema.FS, "catalog.json")
	if err != nil {
		return "", err
	}
	var cat struct {
		SpecVersion string `json:"specVersion"`
	}
	if err := json.Unmarshal(b, &cat); err != nil {
		return "", err
	}
	if cat.SpecVersion == "" {
		return "", fmt.Errorf("embedded schema catalog declares no specification version")
	}
	return cat.SpecVersion, nil
}
