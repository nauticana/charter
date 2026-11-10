package keel

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	charterschema "github.com/nauticana/charter/sdk/schema"
)

// tableColumn, tableForeignKey, and tableDef read the parts of a keel table YAML file the stores need.
type tableColumn struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Nullable bool   `yaml:"nullable"`
	Default  string `yaml:"default"`
}

type tableForeignKey struct {
	Name       string   `yaml:"name"`
	Columns    []string `yaml:"columns"`
	References struct {
		Table   string   `yaml:"table"`
		Columns []string `yaml:"columns"`
	} `yaml:"references"`
}

type tableSequence struct {
	Name   string `yaml:"name"`
	Column string `yaml:"column"`
}

type tableDef struct {
	Name        string            `yaml:"table"`
	Columns     []tableColumn     `yaml:"columns"`
	PrimaryKey  []string          `yaml:"primary_key"`
	ForeignKeys []tableForeignKey `yaml:"foreign_keys"`
	Sequence    *tableSequence    `yaml:"sequence"`
	module      string
}

// column returns the named column; ok is false for a name the table does not define.
func (t *tableDef) column(name string) (tableColumn, bool) {
	for _, c := range t.Columns {
		if c.Name == name {
			return c, true
		}
	}
	return tableColumn{}, false
}

// written lists the columns a store writes: every column except those the database fills by default.
func (t *tableDef) written() []tableColumn {
	out := make([]tableColumn, 0, len(t.Columns))
	for _, c := range t.Columns {
		if c.Default == "" {
			out = append(out, c)
		}
	}
	return out
}

func columnType(c tableColumn) string { return strings.ToUpper(c.Type) }

// catalogModule is one schema module: a group of dependency.yml and the modules its tables reference.
type catalogModule struct {
	Name      string   `yaml:"name"`
	Path      string   `yaml:"path"`
	DependsOn []string `yaml:"depends_on"`
	tables    []string
}

// modulePrefix marks Charter's groups in depends_on; keel's own groups carry no prefix.
const modulePrefix = "charter_"

// tableCatalog is the relational model as the embedded schema modules define it, in creation order.
type tableCatalog struct {
	tables  map[string]*tableDef
	order   []string
	modules []*catalogModule
}

var (
	catalogOnce   sync.Once
	catalogShared *tableCatalog
	catalogErr    error
)

// relationalCatalog parses the embedded modules once.
func relationalCatalog() (*tableCatalog, error) {
	catalogOnce.Do(func() { catalogShared, catalogErr = parseCatalog(charterschema.Modules) })
	return catalogShared, catalogErr
}

func parseCatalog(modules fs.FS) (*tableCatalog, error) {
	var dependency struct {
		Groups []*catalogModule `yaml:"groups"`
	}
	if err := readYAML(modules, "dependency.yml", &dependency); err != nil {
		return nil, err
	}
	c := &tableCatalog{tables: map[string]*tableDef{}}
	for _, g := range dependency.Groups {
		var meta struct {
			Tables []string `yaml:"tables"`
		}
		if err := readYAML(modules, path.Join(g.Path, "ab_meta.yml"), &meta); err != nil {
			return nil, err
		}
		for _, name := range meta.Tables {
			t := &tableDef{}
			if err := readYAML(modules, path.Join(g.Path, name+".yml"), t); err != nil {
				return nil, err
			}
			if t.Name != name || len(t.PrimaryKey) < 2 || t.PrimaryKey[0] != "partner_id" {
				return nil, fmt.Errorf("schema module table %s: name or tenant key is wrong", name)
			}
			t.module = g.Name
			c.tables[name] = t
			c.order = append(c.order, name)
			g.tables = append(g.tables, name)
		}
	}
	c.modules = dependency.Groups
	return c, nil
}

// install returns the tables of the named modules, all modules when none is named. Each module must be known and
// installed with every Charter module it depends on, and charter_object, which every document needs, is required.
func (c *tableCatalog) install(names []string) (map[string]bool, error) {
	byName := map[string]*catalogModule{}
	for _, m := range c.modules {
		byName[m.Name] = m
	}
	if len(names) == 0 {
		for _, m := range c.modules {
			names = append(names, m.Name)
		}
	}
	chosen := map[string]bool{}
	for _, n := range names {
		if byName[n] == nil {
			return nil, fmt.Errorf("%w: %s is not a Charter schema module", ErrModuleNotInstalled, n)
		}
		chosen[n] = true
	}
	if !chosen[modulePrefix+"object"] {
		return nil, fmt.Errorf("%w: every installation needs %sobject", ErrModuleNotInstalled, modulePrefix)
	}
	tables := map[string]bool{}
	for n := range chosen {
		for _, d := range byName[n].DependsOn {
			if strings.HasPrefix(d, modulePrefix) && !chosen[d] {
				return nil, fmt.Errorf("%w: %s needs %s", ErrModuleNotInstalled, n, d)
			}
		}
		for _, t := range byName[n].tables {
			tables[t] = true
		}
	}
	return tables, nil
}

func readYAML(modules fs.FS, name string, into any) error {
	raw, err := fs.ReadFile(modules, name)
	if err != nil {
		return fmt.Errorf("schema module %s: %w", name, err)
	}
	if err := yaml.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("schema module %s: %w", name, err)
	}
	return nil
}

// Query names derive from the table and the operation, so each is generated once from the catalog.
const (
	qObjectByName  = "charter.enterprise_object.by.name"
	qObjectsOfKind = "charter.enterprise_object.of.kind"
	qCurrencies    = "charter.currency.by.code"
)

func qInsert(table string) string           { return "charter." + table + ".insert" }
func qUpdate(table string) string           { return "charter." + table + ".update" }
func qSelectBy(table, column string) string { return "charter." + table + ".by." + column }
func qDeleteBy(table, column string) string { return "charter." + table + ".delete." + column }
func whereAny(column string) string         { return "partner_id = ? AND " + column + " = ANY(?)" }

// selectList reads NUMERIC as text, so no value passes through a float; every other type keeps its driver form.
func selectList(t *tableDef) string {
	out := make([]string, len(t.Columns))
	for i, c := range t.Columns {
		out[i] = c.Name
		if columnType(c) == "NUMERIC" {
			out[i] = "CAST(" + c.Name + " AS TEXT)"
		}
	}
	return strings.Join(out, ", ")
}

// placeholder binds NUMERIC from text for the same reason.
func placeholder(c tableColumn) string {
	if columnType(c) == "NUMERIC" {
		return "CAST(CAST(? AS TEXT) AS NUMERIC)"
	}
	return "?"
}

// lookupColumns names the columns rows are read and deleted by: the first key column after the tenant, and the
// parent of a tree node.
func lookupColumns(t *tableDef) []string {
	out := []string{t.PrimaryKey[1]}
	if _, ok := t.column("parent_id"); ok {
		out = append(out, "parent_id")
	}
	return out
}

// queries generates the named SQL of the installed tables: insert, update by key, and select and delete by lookup
// column.
func (c *tableCatalog) queries(installed map[string]bool) map[string]string {
	q := map[string]string{
		qObjectByName:  "SELECT " + selectList(c.tables["enterprise_object"]) + " FROM enterprise_object WHERE partner_id = ? AND namespace = ? AND identifier = ?",
		qObjectsOfKind: "SELECT " + selectList(c.tables["enterprise_object"]) + " FROM enterprise_object WHERE partner_id = ? AND kind = ?",
		qCurrencies:    "SELECT code, exponent FROM currency WHERE code = ANY(?)",
	}
	for _, name := range c.order {
		if !installed[name] {
			continue
		}
		t := c.tables[name]
		cols, vals := []string{}, []string{}
		returning := ""
		for _, col := range t.written() {
			cols = append(cols, col.Name)
			if t.Sequence != nil && col.Name == t.Sequence.Column {
				vals = append(vals, "nextval('"+t.Sequence.Name+"')")
				returning = " RETURNING " + col.Name
				continue
			}
			vals = append(vals, placeholder(col))
		}
		q[qInsert(name)] = "INSERT INTO " + name + " (" + strings.Join(cols, ", ") + ") VALUES (" + strings.Join(vals, ", ") + ")" + returning
		var sets, keys []string
		for _, col := range t.written() {
			if !slices.Contains(t.PrimaryKey, col.Name) {
				sets = append(sets, col.Name+" = "+placeholder(col))
			}
		}
		for _, k := range t.PrimaryKey {
			keys = append(keys, k+" = ?")
		}
		if len(sets) > 0 {
			q[qUpdate(name)] = "UPDATE " + name + " SET " + strings.Join(sets, ", ") + " WHERE " + strings.Join(keys, " AND ")
		}
		for _, col := range lookupColumns(t) {
			q[qSelectBy(name, col)] = "SELECT " + selectList(t) + " FROM " + name + " WHERE " + whereAny(col)
			q[qDeleteBy(name, col)] = "DELETE FROM " + name + " WHERE " + whereAny(col)
		}
	}
	return q
}

// updatedColumns are the columns qUpdate sets, in order; its key arguments follow them.
func (t *tableDef) updatedColumns() []tableColumn {
	var out []tableColumn
	for _, col := range t.written() {
		if !slices.Contains(t.PrimaryKey, col.Name) {
			out = append(out, col)
		}
	}
	return out
}
