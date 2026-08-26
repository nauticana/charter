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

// CatalogMeta identifies the embedded schema catalog and the specification version its schemas target.
type CatalogMeta struct {
	CatalogVersion string `json:"catalogVersion"`
	SpecVersion    string `json:"specVersion"`
}

func (Catalog) Meta() (CatalogMeta, error) {
	var meta CatalogMeta
	b, err := fs.ReadFile(schema.FS, "catalog.json")
	if err != nil {
		return meta, err
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		return meta, err
	}
	b, err = fs.ReadFile(schema.FS, "common.schema.json")
	if err != nil {
		return meta, err
	}
	var common struct {
		SpecVersion string `json:"x-charter-spec-version"`
	}
	if err := json.Unmarshal(b, &common); err != nil {
		return meta, err
	}
	meta.SpecVersion = common.SpecVersion
	if meta.CatalogVersion == "" || meta.SpecVersion == "" {
		return meta, fmt.Errorf("embedded schemas declare no catalog or specification version")
	}
	return meta, nil
}
