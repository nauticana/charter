package keel

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nauticana/keel/schema"
)

// TestSchemaModuleCompilesOverKeelTables checks the module the stores query against keel's own table definitions.
func TestSchemaModuleCompilesOverKeelTables(t *testing.T) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/nauticana/keel").Output()
	if err != nil {
		t.Fatalf("locate keel: %v", err)
	}
	keel := filepath.Join(strings.TrimSpace(string(out)), "schema")
	s, err := schema.ParseDirs([]string{filepath.Join(keel, "core"), filepath.Join(keel, "geo"), filepath.Join(keel, "tenant_management"), "schema"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"charter_document", "charter_document_revision", "charter_human_account"} {
		if s.GetTable(table) == nil {
			t.Errorf("table %s is missing from the module", table)
		}
	}
}
