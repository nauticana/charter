package keel

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	kmodel "github.com/nauticana/keel/model"
	"github.com/nauticana/keel/port"
	"gopkg.in/yaml.v3"

	charterschema "github.com/nauticana/charter/sdk/schema"
)

// memTable is a table as the fake enforces it: the schema module's columns, keys, unique indexes, foreign keys, and
// the code lists its CHECKs name.
type memTable struct {
	def     *tableDef
	uniques []memUnique
	codes   map[string][]string
}

type memUnique struct {
	name  string
	cols  []string
	where []memPredicate
}

type memPredicate struct {
	col, op, literal string
}

var (
	codeCheck = regexp.MustCompile(`^(\w+) IN \(((?:'[A-Z]+'(?:, )?)+)\)$`)
	predicate = regexp.MustCompile(`^(\w+) (IS NOT NULL|IS NULL|=|<>) ?(?:'([^']*)')?$`)
)

// memSchema parses the embedded modules with their indexes and checks, plus the three keel tables they reference.
func memSchema(t *testing.T) map[string]*memTable {
	t.Helper()
	out := map[string]*memTable{
		"business_partner": {def: &tableDef{Name: "business_partner", PrimaryKey: []string{"id"},
			Columns: []tableColumn{{Name: "id", Type: "BIGINT"}, {Name: "caption", Type: "VARCHAR(200)"}}}},
		"partner_user": {def: &tableDef{Name: "partner_user", PrimaryKey: []string{"partner_id", "user_id", "begda"},
			Columns: []tableColumn{{Name: "partner_id", Type: "BIGINT"}, {Name: "user_id", Type: "BIGINT"},
				{Name: "begda", Type: "TIMESTAMP"}, {Name: "endda", Type: "TIMESTAMP", Nullable: true}}}},
		"currency": {def: &tableDef{Name: "currency", PrimaryKey: []string{"code"},
			Columns: []tableColumn{{Name: "code", Type: "CHAR(3)"}, {Name: "caption", Type: "VARCHAR(80)"}, {Name: "exponent", Type: "SMALLINT"}}}},
	}
	cat, err := relationalCatalog()
	if err != nil {
		t.Fatal(err)
	}
	err = fs.WalkDir(charterschema.Modules, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || path.Base(p) == "ab_meta.yml" || path.Base(p) == "dependency.yml" || strings.HasPrefix(p, "seed/") {
			return err
		}
		var extra struct {
			Table   string `yaml:"table"`
			Indexes []struct {
				Name    string   `yaml:"name"`
				Columns []string `yaml:"columns"`
				Unique  bool     `yaml:"unique"`
				Where   string   `yaml:"where"`
			} `yaml:"indexes"`
			Checks []struct {
				Expression string `yaml:"expression"`
			} `yaml:"checks"`
		}
		raw, err := fs.ReadFile(charterschema.Modules, p)
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(raw, &extra); err != nil {
			return err
		}
		mt := &memTable{def: cat.tables[extra.Table], codes: map[string][]string{}}
		for _, idx := range extra.Indexes {
			if !idx.Unique {
				continue
			}
			u := memUnique{name: idx.Name, cols: idx.Columns}
			for _, part := range strings.Split(idx.Where, " AND ") {
				if part == "" {
					continue
				}
				m := predicate.FindStringSubmatch(part)
				if m == nil {
					return fmt.Errorf("%s: unsupported predicate %q", idx.Name, part)
				}
				u.where = append(u.where, memPredicate{col: m[1], op: m[2], literal: m[3]})
			}
			mt.uniques = append(mt.uniques, u)
		}
		for _, c := range extra.Checks {
			if m := codeCheck.FindStringSubmatch(c.Expression); m != nil {
				for _, code := range strings.Split(m[2], ", ") {
					mt.codes[m[1]] = append(mt.codes[m[1]], strings.Trim(code, "'"))
				}
			}
		}
		out[extra.Table] = mt
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

type memState struct {
	rows      map[string][]map[string]any
	sequences map[string]int64
}

func (s memState) clone() memState {
	c := memState{rows: map[string][]map[string]any{}, sequences: map[string]int64{}}
	for t, rows := range s.rows {
		for _, r := range rows {
			copied := make(map[string]any, len(r))
			for k, v := range r {
				copied[k] = v
			}
			c.rows[t] = append(c.rows[t], copied)
		}
	}
	for k, v := range s.sequences {
		c.sequences[k] = v
	}
	return c
}

// memoryDB executes the named SQL the stores generate over in-memory tables, enforcing the schema modules' keys, unique
// indexes, foreign keys, NOT NULL columns, column widths, and code-list CHECKs; a transaction works on a copy that
// commit publishes.
type memoryDB struct {
	port.DatabaseRepository
	tables     map[string]*memTable
	state      memState
	failQuery  string
	failCommit bool
	rollbacks  int
	calls      map[string]int
}

// newMemoryDB holds the tables of the named modules, every module when none is named, and keel's referenced tables.
func newMemoryDB(t *testing.T, modules ...string) *memoryDB {
	tables := memSchema(t)
	m, err := relational()
	if err != nil {
		t.Fatal(err)
	}
	installed, err := m.cat.install(modules)
	if err != nil {
		t.Fatal(err)
	}
	for name := range tables {
		if _, charter := m.cat.tables[name]; charter && !installed[name] {
			delete(tables, name)
		}
	}
	db := &memoryDB{tables: tables, calls: map[string]int{}, state: memState{rows: map[string][]map[string]any{}, sequences: map[string]int64{}}}
	db.state.rows["business_partner"] = []map[string]any{{"id": partner, "caption": "Harbor"}, {"id": partner + 1, "caption": "Other"}}
	db.state.rows["currency"] = []map[string]any{{"code": "EUR", "caption": "Euro", "exponent": int64(2)},
		{"code": "USD", "caption": "US dollar", "exponent": int64(2)}, {"code": "JPY", "caption": "Yen", "exponent": int64(0)}}
	return db
}

func (db *memoryDB) GetQueryService(_ context.Context, queries map[string]string) port.QueryService {
	return &memoryQueries{db: db, state: &db.state, sql: queries}
}

func (db *memoryDB) BeginTx(_ context.Context, queries map[string]string) (port.TxQueryService, error) {
	state := db.state.clone()
	return &memoryTx{memoryQueries{db: db, state: &state, sql: queries}}, nil
}

// count returns the rows of a table, for assertions.
func (db *memoryDB) count(table string) int { return len(db.state.rows[table]) }

type memoryQueries struct {
	db    *memoryDB
	state *memState
	sql   map[string]string
}

type memoryTx struct{ memoryQueries }

func (tx *memoryTx) Commit(context.Context) error {
	if tx.db.failCommit {
		return errors.New("commit failed")
	}
	tx.db.state = *tx.state
	return nil
}

func (tx *memoryTx) Rollback(context.Context) error {
	tx.db.rollbacks++
	return nil
}

func (q *memoryQueries) GenID() int64 { return 0 }

func (q *memoryQueries) Query(_ context.Context, name string, args ...any) (*kmodel.QueryResult, error) {
	q.db.calls[name]++
	if name == q.db.failQuery {
		return nil, fmt.Errorf("query %s failed", name)
	}
	if h, ok := humanAccountHandlers[name]; ok {
		return h(q, args)
	}
	sql, ok := q.sql[name]
	if !ok {
		return nil, fmt.Errorf("unknown query %s", name)
	}
	stmt := strings.TrimSuffix(strings.Join(strings.Fields(sql), " "), " FOR UPDATE")
	ex := &memExec{q: q, args: args}
	var res *kmodel.QueryResult
	var err error
	switch {
	case strings.HasPrefix(stmt, "INSERT INTO "):
		res, err = ex.insert(stmt)
	case strings.HasPrefix(stmt, "SELECT "):
		res, err = ex.selectRows(stmt)
	case strings.HasPrefix(stmt, "UPDATE "):
		res, err = ex.update(stmt)
	case strings.HasPrefix(stmt, "DELETE FROM "):
		res, err = ex.delete(stmt)
	default:
		err = fmt.Errorf("unsupported statement %s", stmt)
	}
	if err == nil && ex.next != len(args) {
		err = fmt.Errorf("%s used %d of %d arguments", name, ex.next, len(args))
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return res, nil
}

type memExec struct {
	q    *memoryQueries
	args []any
	next int
}

func (ex *memExec) arg() (any, error) {
	if ex.next >= len(ex.args) {
		return nil, errors.New("too few arguments")
	}
	v := ex.args[ex.next]
	ex.next++
	return v, nil
}

var (
	insertStmt = regexp.MustCompile(`^INSERT INTO (\w+) \((.*?)\) VALUES \((.*)\)(?: RETURNING (\w+))?$`)
	selectStmt = regexp.MustCompile(`^SELECT (.*?) FROM (\w+) WHERE (.*)$`)
	updateStmt = regexp.MustCompile(`^UPDATE (\w+) SET (.*) WHERE (.*)$`)
	deleteStmt = regexp.MustCompile(`^DELETE FROM (\w+) WHERE (.*)$`)
	castColumn = regexp.MustCompile(`^CAST\((\w+) AS TEXT\)$`)
	nextval    = regexp.MustCompile(`^nextval\('(\w+)'\)$`)
	condition  = regexp.MustCompile(`^(\w+) (=|<>) (\?|ANY\(\?\)|'[^']*')$`)
)

// splitTop splits on sep outside parentheses.
func splitTop(s, sep string) []string {
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth == 0 && strings.HasPrefix(s[i:], sep) {
			out = append(out, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	return append(out, s[start:])
}

func (ex *memExec) table(name string) (*memTable, error) {
	t, ok := ex.q.db.tables[name]
	if !ok || t.def == nil {
		return nil, fmt.Errorf("no table %s", name)
	}
	return t, nil
}

// value reads one bound value or sequence call and checks it against the column.
func (ex *memExec) value(t *memTable, col, expr string) (any, error) {
	c, ok := t.def.column(col)
	if !ok {
		return nil, fmt.Errorf("no column %s.%s", t.def.Name, col)
	}
	if m := nextval.FindStringSubmatch(expr); m != nil {
		if t.def.Sequence == nil || t.def.Sequence.Name != m[1] {
			return nil, fmt.Errorf("%s has no sequence %s", t.def.Name, m[1])
		}
		ex.q.state.sequences[m[1]]++
		return ex.q.state.sequences[m[1]], nil
	}
	numeric := expr == "CAST(CAST(? AS TEXT) AS NUMERIC)"
	if expr != "?" && !numeric {
		return nil, fmt.Errorf("unsupported value %s", expr)
	}
	if numeric != (columnType(c) == "NUMERIC") {
		return nil, fmt.Errorf("%s.%s binds %s", t.def.Name, col, expr)
	}
	v, err := ex.arg()
	if err != nil {
		return nil, err
	}
	return checkValue(c, v)
}

// checkValue is the column's type check: the Go type a driver argument must have, and its width.
func checkValue(c tableColumn, v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	typ := columnType(c)
	bad := fmt.Errorf("column %s %s cannot hold %T %v", c.Name, typ, v, v)
	switch {
	case typ == "BIGINT" || typ == "INT" || typ == "INTEGER" || typ == "SMALLINT":
		switch x := v.(type) {
		case int64:
			return x, nil
		case int:
			return int64(x), nil
		}
		return nil, bad
	case typ == "BOOLEAN":
		if _, ok := v.(bool); ok {
			return v, nil
		}
		return nil, bad
	case typ == "NUMERIC":
		s, ok := v.(string)
		if _, valid := new(big.Rat).SetString(s); !ok || !valid {
			return nil, bad
		}
		return s, nil
	case typ == "TIMESTAMP" || typ == "DATE":
		x, ok := v.(time.Time)
		if !ok {
			return nil, bad
		}
		if typ == "DATE" {
			return time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, time.UTC), nil
		}
		return x.UTC(), nil
	case typ == "TEXT":
		if _, ok := v.(string); ok {
			return v, nil
		}
		return nil, bad
	}
	s, ok := v.(string)
	if !ok {
		return nil, bad
	}
	if open := strings.Index(typ, "("); open > 0 {
		width, _ := strconv.Atoi(strings.TrimSuffix(typ[open+1:], ")"))
		if utf8.RuneCountInString(s) > width {
			return nil, fmt.Errorf("column %s %s cannot hold %d characters", c.Name, typ, utf8.RuneCountInString(s))
		}
	}
	return s, nil
}

func (ex *memExec) insert(stmt string) (*kmodel.QueryResult, error) {
	m := insertStmt.FindStringSubmatch(stmt)
	if m == nil {
		return nil, fmt.Errorf("unsupported insert %s", stmt)
	}
	t, err := ex.table(m[1])
	if err != nil {
		return nil, err
	}
	cols, vals := strings.Split(m[2], ", "), splitTop(m[3], ", ")
	if len(cols) != len(vals) {
		return nil, fmt.Errorf("%s: %d columns, %d values", t.def.Name, len(cols), len(vals))
	}
	row := map[string]any{}
	for i, col := range cols {
		v, err := ex.value(t, col, vals[i])
		if err != nil {
			return nil, err
		}
		row[col] = v
	}
	for _, c := range t.def.Columns {
		if _, set := row[c.Name]; !set && c.Default != "" {
			row[c.Name] = time.Now().UTC()
		}
	}
	if err := ex.checkRow(t, row, -1); err != nil {
		return nil, err
	}
	ex.q.state.rows[t.def.Name] = append(ex.q.state.rows[t.def.Name], row)
	if m[4] != "" {
		return &kmodel.QueryResult{Rows: [][]any{{row[m[4]]}}}, nil
	}
	return &kmodel.QueryResult{}, nil
}

// where matches rows against conditions binding arguments left to right.
func (ex *memExec) where(t *memTable, clause string) (func(map[string]any) bool, error) {
	type test func(map[string]any) bool
	var tests []test
	for _, part := range splitTop(clause, " AND ") {
		m := condition.FindStringSubmatch(part)
		if m == nil {
			return nil, fmt.Errorf("unsupported condition %q", part)
		}
		col, op, operand := m[1], m[2], m[3]
		if _, ok := t.def.column(col); !ok {
			return nil, fmt.Errorf("no column %s.%s", t.def.Name, col)
		}
		switch {
		case strings.HasPrefix(operand, "'"):
			literal := strings.Trim(operand, "'")
			tests = append(tests, func(r map[string]any) bool { return (r[col] == literal) == (op == "=") })
		case operand == "?":
			v, err := ex.arg()
			if err != nil {
				return nil, err
			}
			want := rowKey(normalizeArg(v))
			tests = append(tests, func(r map[string]any) bool { return r[col] != nil && rowKey(r[col]) == want })
		default:
			v, err := ex.arg()
			if err != nil {
				return nil, err
			}
			set := map[string]bool{}
			switch list := v.(type) {
			case []int64:
				for _, x := range list {
					set[rowKey(x)] = true
				}
			case []string:
				for _, x := range list {
					set[rowKey(x)] = true
				}
			default:
				return nil, fmt.Errorf("ANY over %T", v)
			}
			tests = append(tests, func(r map[string]any) bool { return r[col] != nil && set[rowKey(r[col])] })
		}
	}
	return func(r map[string]any) bool {
		for _, test := range tests {
			if !test(r) {
				return false
			}
		}
		return true
	}, nil
}

func normalizeArg(v any) any {
	switch x := v.(type) {
	case int:
		return int64(x)
	case time.Time:
		return x.UTC()
	}
	return v
}

func (ex *memExec) selectRows(stmt string) (*kmodel.QueryResult, error) {
	m := selectStmt.FindStringSubmatch(stmt)
	if m == nil {
		return nil, fmt.Errorf("unsupported select %s", stmt)
	}
	t, err := ex.table(m[2])
	if err != nil {
		return nil, err
	}
	var cols []string
	for _, item := range splitTop(m[1], ", ") {
		if c := castColumn.FindStringSubmatch(item); c != nil {
			col, _ := t.def.column(c[1])
			if columnType(col) != "NUMERIC" {
				return nil, fmt.Errorf("%s casts %s", t.def.Name, c[1])
			}
			item = c[1]
		} else if col, ok := t.def.column(item); ok && columnType(col) == "NUMERIC" {
			return nil, fmt.Errorf("%s reads NUMERIC %s without a text cast", t.def.Name, item)
		}
		if _, ok := t.def.column(item); !ok {
			return nil, fmt.Errorf("no column %s.%s", t.def.Name, item)
		}
		cols = append(cols, item)
	}
	match, err := ex.where(t, m[3])
	if err != nil {
		return nil, err
	}
	res := &kmodel.QueryResult{}
	for _, r := range ex.q.state.rows[t.def.Name] {
		if match(r) {
			out := make([]any, len(cols))
			for i, c := range cols {
				out[i] = r[c]
			}
			res.Rows = append(res.Rows, out)
		}
	}
	return res, nil
}

func (ex *memExec) update(stmt string) (*kmodel.QueryResult, error) {
	m := updateStmt.FindStringSubmatch(stmt)
	if m == nil {
		return nil, fmt.Errorf("unsupported update %s", stmt)
	}
	t, err := ex.table(m[1])
	if err != nil {
		return nil, err
	}
	sets := map[string]any{}
	for _, assignment := range splitTop(m[2], ", ") {
		col, expr, ok := strings.Cut(assignment, " = ")
		if !ok {
			return nil, fmt.Errorf("unsupported assignment %s", assignment)
		}
		v, err := ex.value(t, col, expr)
		if err != nil {
			return nil, err
		}
		sets[col] = v
	}
	match, err := ex.where(t, m[3])
	if err != nil {
		return nil, err
	}
	rows := ex.q.state.rows[t.def.Name]
	for i, r := range rows {
		if !match(r) {
			continue
		}
		updated := make(map[string]any, len(r))
		for k, v := range r {
			updated[k] = v
		}
		for k, v := range sets {
			updated[k] = v
		}
		if err := ex.checkRow(t, updated, i); err != nil {
			return nil, err
		}
		rows[i] = updated
	}
	return &kmodel.QueryResult{}, ex.checkReferences(t.def.Name)
}

func (ex *memExec) delete(stmt string) (*kmodel.QueryResult, error) {
	m := deleteStmt.FindStringSubmatch(stmt)
	if m == nil {
		return nil, fmt.Errorf("unsupported delete %s", stmt)
	}
	t, err := ex.table(m[1])
	if err != nil {
		return nil, err
	}
	match, err := ex.where(t, m[2])
	if err != nil {
		return nil, err
	}
	ex.q.state.rows[t.def.Name] = slices.DeleteFunc(ex.q.state.rows[t.def.Name], match)
	return &kmodel.QueryResult{}, ex.checkReferences(t.def.Name)
}

// checkRow enforces NOT NULL, code lists, the key, unique indexes, and foreign keys on a written row; self is the
// row's index when it replaces an existing row.
func (ex *memExec) checkRow(t *memTable, row map[string]any, self int) error {
	for _, c := range t.def.Columns {
		v := row[c.Name]
		if v == nil && !c.Nullable {
			return fmt.Errorf("%s.%s is NOT NULL", t.def.Name, c.Name)
		}
		if codes, ok := t.codes[c.Name]; ok && v != nil && !slices.Contains(codes, v.(string)) {
			return fmt.Errorf("%s.%s CHECK refuses %v", t.def.Name, c.Name, v)
		}
	}
	keys := append([]memUnique{{name: "pk", cols: t.def.PrimaryKey}}, t.uniques...)
	for _, u := range keys {
		if !applies(u, row) {
			continue
		}
		want := projection(row, u.cols)
		for i, other := range ex.q.state.rows[t.def.Name] {
			if i != self && applies(u, other) && projection(other, u.cols) == want {
				return fmt.Errorf("duplicate key %s on %s %s", u.name, t.def.Name, want)
			}
		}
	}
	for _, fk := range t.def.ForeignKeys {
		if !ex.parentExists(fk, row) {
			return fmt.Errorf("foreign key %s on %s: no %s row", fk.Name, t.def.Name, fk.References.Table)
		}
	}
	return nil
}

func applies(u memUnique, row map[string]any) bool {
	for _, c := range u.cols {
		if row[c] == nil {
			return false
		}
	}
	for _, p := range u.where {
		v := row[p.col]
		switch p.op {
		case "IS NULL":
			if v != nil {
				return false
			}
		case "IS NOT NULL":
			if v == nil {
				return false
			}
		case "=":
			if v != p.literal {
				return false
			}
		case "<>":
			if v == p.literal {
				return false
			}
		}
	}
	return true
}

func projection(row map[string]any, cols []string) string {
	values := make([]any, len(cols))
	for i, c := range cols {
		values[i] = row[c]
	}
	return rowKey(values...)
}

func (ex *memExec) parentExists(fk tableForeignKey, row map[string]any) bool {
	values := make([]any, len(fk.Columns))
	for i, c := range fk.Columns {
		if row[c] == nil {
			return true
		}
		values[i] = row[c]
	}
	want := rowKey(values...)
	for _, parent := range ex.q.state.rows[fk.References.Table] {
		if projection(parent, fk.References.Columns) == want {
			return true
		}
	}
	return false
}

// checkReferences refuses a statement that leaves a row referencing a row of table that no longer exists.
func (ex *memExec) checkReferences(table string) error {
	for name, t := range ex.q.db.tables {
		for _, fk := range t.def.ForeignKeys {
			if fk.References.Table != table {
				continue
			}
			for _, row := range ex.q.state.rows[name] {
				if !ex.parentExists(fk, row) {
					return fmt.Errorf("foreign key %s on %s: its %s row is gone", fk.Name, name, table)
				}
			}
		}
	}
	return nil
}

// humanAccountHandlers run the account queries whose joins and disjunctions the statement grammar does not cover.
var humanAccountHandlers = map[string]func(q *memoryQueries, args []any) (*kmodel.QueryResult, error){
	qAccountOverlaps: func(q *memoryQueries, args []any) (*kmodel.QueryResult, error) {
		res := &kmodel.QueryResult{}
		open := args[3].(bool)
		endda, finite := args[4].(time.Time)
		for _, a := range q.state.rows["human_identity_partner_user"] {
			if a["partner_id"] == args[0] && (a["human_identity_id"] == args[1] || a["user_id"] == args[2]) &&
				(open || finite && a["begda"].(time.Time).Before(endda)) && (a["endda"] == nil || a["endda"].(time.Time).After(args[5].(time.Time))) {
				res.Rows = append(res.Rows, []any{a["human_identity_id"], a["user_id"], a["begda"], a["endda"]})
			}
		}
		return res, nil
	},
	qAccountOpenAt: func(q *memoryQueries, args []any) (*kmodel.QueryResult, error) {
		res := &kmodel.QueryResult{}
		for _, a := range q.state.rows["human_identity_partner_user"] {
			if a["partner_id"] == args[0] && a["human_identity_id"] == args[1] && a["begda"].(time.Time).Before(args[2].(time.Time)) &&
				(a["endda"] == nil || a["endda"].(time.Time).After(args[3].(time.Time))) {
				res.Rows = append(res.Rows, []any{a["begda"]})
			}
		}
		return res, nil
	},
	qHumanOfUser: func(q *memoryQueries, args []any) (*kmodel.QueryResult, error) {
		return effectiveLinks(q, args[0].(int64), args[len(args)-1].(time.Time), func(a, o map[string]any) ([]any, bool) {
			return []any{o["namespace"], o["identifier"]}, a["user_id"] == args[1]
		})
	},
	qUserOfHuman: func(q *memoryQueries, args []any) (*kmodel.QueryResult, error) {
		return effectiveLinks(q, args[0].(int64), args[len(args)-1].(time.Time), func(a, o map[string]any) ([]any, bool) {
			return []any{a["user_id"]}, o["namespace"] == args[1] && o["identifier"] == args[2] && o["kind"] == humanKindCode
		})
	},
}

func effectiveLinks(q *memoryQueries, partnerID int64, at time.Time, pick func(a, o map[string]any) ([]any, bool)) (*kmodel.QueryResult, error) {
	res := &kmodel.QueryResult{}
	for _, a := range q.state.rows["human_identity_partner_user"] {
		if a["partner_id"] != partnerID || a["begda"].(time.Time).After(at) || (a["endda"] != nil && !a["endda"].(time.Time).After(at)) {
			continue
		}
		for _, p := range q.state.rows["partner_user"] {
			if p["partner_id"] != partnerID || p["user_id"] != a["user_id"] || !p["begda"].(time.Time).Equal(a["partner_user_begda"].(time.Time)) ||
				(p["endda"] != nil && !p["endda"].(time.Time).After(at)) {
				continue
			}
			for _, o := range q.state.rows["enterprise_object"] {
				if o["partner_id"] == partnerID && o["id"] == a["human_identity_id"] {
					if row, ok := pick(a, o); ok {
						res.Rows = append(res.Rows, row)
					}
				}
			}
		}
	}
	return res, nil
}
