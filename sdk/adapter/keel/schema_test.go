package keel

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nauticana/keel/schema"
	"github.com/nauticana/keel/schema/dialect"
)

// TestSchemaModuleCompilesOverKeelTables checks the module the stores query against keel's own table definitions.
func TestSchemaModuleCompilesOverKeelTables(t *testing.T) {
	s := moduleSchema(t)
	for _, table := range []string{"charter_document", "charter_document_revision", "charter_human_account"} {
		if s.GetTable(table) == nil {
			t.Errorf("table %s is missing from the module", table)
		}
	}
}

// TestHumanAccountPeriodsCannotOverlap checks that PostgreSQL itself refuses overlapping links per human and per user.
func TestHumanAccountPeriodsCannotOverlap(t *testing.T) {
	ddl := (&dialect.PgSQL{}).GenerateTable(moduleSchema(t).GetTable("charter_human_account"))
	for _, want := range []string{
		"CREATE EXTENSION IF NOT EXISTS btree_gist;",
		"CONSTRAINT charter_human_account_human_no_overlap EXCLUDE USING gist (partner_id WITH =, human_namespace WITH =, human_id WITH =, tsrange(begda, endda) WITH &&)",
		"CONSTRAINT charter_human_account_user_no_overlap EXCLUDE USING gist (partner_id WITH =, user_id WITH =, tsrange(begda, endda) WITH &&)",
	} {
		if !strings.Contains(ddl, want) {
			t.Errorf("charter_human_account DDL lacks %q:\n%s", want, ddl)
		}
	}
}

func moduleSchema(t *testing.T) *schema.Schema {
	t.Helper()
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
	return s
}
