package keel

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/nauticana/keel/schema"
)

const charterTableCount = 166

// charterSchema parses only Charter's modules, without keel's tables, so tests can tell Charter's tables apart.
func charterSchema(t *testing.T) *schema.Schema {
	t.Helper()
	cat, err := relationalCatalog()
	if err != nil {
		t.Fatal(err)
	}
	var dirs []string
	for _, m := range cat.modules {
		dirs = append(dirs, filepath.Join("..", "..", "schema", m.Path))
	}
	s, err := schema.ParseDirs(dirs)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func charterSeeds(t *testing.T) []*schema.SeedFile {
	t.Helper()
	seeds, err := schema.ParseSeedPath(filepath.Join("..", "..", "schema", "seed"))
	if err != nil {
		t.Fatal(err)
	}
	return seeds
}

// TestRelationalModelTables checks that every table of schema/database.md exists, is keyed by the tenant, and that
// every foreign key name is unique across the whole schema, since keel derives REST nesting from those names.
func TestRelationalModelTables(t *testing.T) {
	s := charterSchema(t)
	if len(s.Tables) != charterTableCount {
		t.Errorf("expected %d tables, found %d", charterTableCount, len(s.Tables))
	}
	merged := moduleSchema(t)
	if err := schema.ValidateSeeds(charterSeeds(t), merged); err != nil {
		t.Error(err)
	}
	fkOwner := map[string]string{}
	for _, table := range s.Tables {
		if len(table.PrimaryKey) == 0 || table.PrimaryKey[0] != "partner_id" {
			t.Errorf("%s: primary key does not lead with partner_id", table.Name)
		}
		for _, fk := range table.ForeignKeys {
			if other, dup := fkOwner[fk.Name]; dup {
				t.Errorf("foreign key %s is declared on both %s and %s", fk.Name, other, table.Name)
			}
			fkOwner[fk.Name] = table.Name
			if fk.References.Table == "currency" {
				continue
			}
			if fk.Columns[0] != "partner_id" {
				t.Errorf("%s.%s: foreign key does not lead with partner_id", table.Name, fk.Name)
			}
		}
		for _, idx := range table.Indexes {
			if len(idx.Name) > 63 {
				t.Errorf("%s: index name %s exceeds 63 characters", table.Name, idx.Name)
			}
		}
	}
	for _, mergedTable := range merged.Tables {
		if s.GetTable(mergedTable.Name) != nil {
			continue
		}
		for _, fk := range mergedTable.ForeignKeys {
			if other, dup := fkOwner[fk.Name]; dup {
				t.Errorf("foreign key %s on %s collides with %s on %s", fk.Name, mergedTable.Name, fk.Name, other)
			}
		}
	}
}

var codeLiteral = regexp.MustCompile(`'([A-Z]{1,2})'`)

// TestCodeColumnsMatchConstantValues checks that every CHAR(1) and CHAR(2) column has a constant_lookup row and a
// CHECK listing exactly the constant_value codes of its domain, so a new code or column cannot drift.
func TestCodeColumnsMatchConstantValues(t *testing.T) {
	s := charterSchema(t)
	values := map[string][]string{}
	lookups := map[string]string{}
	for _, sf := range charterSeeds(t) {
		for _, st := range sf.Seeds {
			col := func(row []any, name string) string {
				for i, c := range st.Columns {
					if c == name {
						return row[i].(string)
					}
				}
				t.Fatalf("seed table %s has no column %s", st.Table, name)
				return ""
			}
			for _, row := range st.Rows {
				switch st.Table {
				case "constant_value":
					values[col(row, "constant_id")] = append(values[col(row, "constant_id")], col(row, "value"))
				case "constant_lookup":
					lookups[col(row, "table_name")+"."+col(row, "column_name")] = col(row, "constant_id")
				}
			}
		}
	}
	seen := map[string]bool{}
	for _, table := range s.Tables {
		for _, column := range table.Columns {
			upper := strings.ToUpper(column.Type)
			if upper != "CHAR(1)" && upper != "CHAR(2)" {
				continue
			}
			key := table.Name + "." + column.Name
			constant, ok := lookups[key]
			if !ok {
				t.Errorf("%s has no constant_lookup row", key)
				continue
			}
			seen[key] = true
			want := append([]string(nil), values[constant]...)
			sort.Strings(want)
			var got []string
			for _, chk := range table.Checks {
				if chk.Name == table.Name+"_"+column.Name+"_ck" {
					for _, m := range codeLiteral.FindAllStringSubmatch(chk.Expression, -1) {
						got = append(got, m[1])
					}
				}
			}
			sort.Strings(got)
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Errorf("%s: CHECK lists %v, constant %s has %v", key, got, constant, want)
			}
		}
	}
	for key := range lookups {
		if !seen[key] {
			t.Errorf("constant_lookup names %s, which is not a code column", key)
		}
	}
}
