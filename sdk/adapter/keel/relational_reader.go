package keel

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nauticana/keel/common"
	"github.com/nauticana/keel/port"

	"github.com/nauticana/charter/sdk/model"
)

// tableIO says how a table's columns read back into logical values.
type tableIO struct {
	spec     *tableSpec
	def      *tableDef
	refs     map[string]bool   // columns holding an object id
	pins     map[string]string // version column to the column holding its definition
	trees    map[string]byte   // 'j' json_value, 's' json_schema_node
	refArcs  map[string]string // column to the pseudo column of its restricted idRef
	approved bool
	measured bool
	split    *splitIO
}

// splitIO is a table one member kind of a split list lands in.
type splitIO struct {
	pseudo string
	owner  []string
	col    string
}

type relationalModel struct {
	cat *tableCatalog
	io  map[string]*tableIO
}

// installation is the part of the model a database holds: the tables of its modules and their named SQL.
type installation struct {
	*relationalModel
	tables  map[string]bool
	queries map[string]string
}

// install selects the named modules; none names every module.
func (m *relationalModel) install(modules []string) (*installation, error) {
	tables, err := m.cat.install(modules)
	if err != nil {
		return nil, err
	}
	return &installation{relationalModel: m, tables: tables, queries: m.cat.queries(tables)}, nil
}

// kind returns the mapping of an installed kind.
func (in *installation) kind(k model.Kind) (*kindSpec, error) {
	spec, ok := kindSpecs[k]
	if !ok {
		return nil, fmt.Errorf("%w: kind %s", ErrUnmappedProperty, k)
	}
	if !in.tables[spec.root.name] {
		return nil, fmt.Errorf("%w: %s needs %s", ErrModuleNotInstalled, k, in.cat.tables[spec.root.name].module)
	}
	return spec, nil
}

// admits refuses a document with rows in a table its installation lacks, naming the module that holds it.
func (in *installation) admits(d *docRows) error {
	var missing []string
	d.spec.tables(func(t *tableSpec, l *listSpec) {
		if len(d.rows[t.name]) == 0 {
			return
		}
		names := []string{t.name}
		if l != nil {
			names = names[:0]
			for _, target := range l.targets {
				names = append(names, target.table)
			}
		}
		for _, n := range names {
			if !in.tables[n] {
				missing = append(missing, n+" of "+in.cat.tables[n].module)
			}
		}
	})
	if len(missing) > 0 {
		return fmt.Errorf("%w: %s %s needs %v", ErrModuleNotInstalled, d.spec.kind, d.key(), missing)
	}
	return nil
}

var (
	modelOnce   sync.Once
	modelShared *relationalModel
	modelErr    error
)

// relational parses the embedded schema modules and checks every kind mapping against them once.
func relational() (*relationalModel, error) {
	modelOnce.Do(func() { modelShared, modelErr = buildRelational() })
	return modelShared, modelErr
}

func buildRelational() (*relationalModel, error) {
	cat, err := relationalCatalog()
	if err != nil {
		return nil, err
	}
	m := &relationalModel{cat: cat, io: map[string]*tableIO{}}
	for _, spec := range kindSpecs {
		var problems []string
		spec.tables(func(t *tableSpec, l *listSpec) {
			if l != nil {
				for _, target := range l.targets {
					def := cat.tables[target.table]
					if def == nil {
						problems = append(problems, "no table "+target.table)
						continue
					}
					m.io[target.table] = &tableIO{spec: t, def: def, refs: map[string]bool{target.col: true},
						split: &splitIO{pseudo: t.name, owner: t.owner, col: target.col}}
				}
				return
			}
			def := cat.tables[t.name]
			if def == nil {
				problems = append(problems, "no table "+t.name)
				return
			}
			io := &tableIO{spec: t, def: def, refs: map[string]bool{}, pins: map[string]string{}, trees: map[string]byte{}, refArcs: map[string]string{}}
			for _, f := range t.fields {
				switch f.kind {
				case fRef, fObject, fActor:
					io.refs[f.col] = true
				case fArc:
					for _, a := range f.arcs {
						io.refs[a.col] = true
					}
				case fRefArc:
					for _, a := range f.arcs {
						io.refs[a.col], io.refArcs[a.col] = true, "#"+f.path
					}
				case fPin:
					io.refs[f.col], io.pins[f.verCol] = true, f.col
				case fJSON:
					io.trees[f.col] = 'j'
				case fSchema:
					io.trees[f.col] = 's'
				case fApproved:
					io.approved = true
					io.refs["approved_capability_id"] = true
				case fMeasured:
					io.measured = true
				}
				for _, c := range f.columns() {
					if _, ok := def.column(c); !ok {
						problems = append(problems, t.name+"."+c)
					}
				}
			}
			for _, c := range append(append(append([]string{}, t.owner...), t.key...), t.order, t.prose, t.form) {
				if _, ok := def.column(c); c != "" && !ok && !strings.HasPrefix(c, "#") {
					problems = append(problems, t.name+"."+c)
				}
			}
			m.io[t.name] = io
		})
		if len(problems) > 0 {
			return nil, fmt.Errorf("%s mapping names columns the schema modules lack: %v", spec.kind, problems)
		}
	}
	return m, nil
}

// storedObject is one enterprise_object row and the version of it to read.
type storedObject struct {
	id         int64
	key        objKey
	code       string
	enterprise *int64
	version    int64
}

// loadedDocument is a stored document's logical rows and what its export derives from other rows.
type loadedDocument struct {
	object     storedObject
	rows       lrows
	enterprise *objKey
	agent      *objKey
	kindOf     map[objKey]model.Kind
	exponent   map[string]int64
}

// relationalReader reads documents through a query service: the store's own, or a write transaction's.
type relationalReader struct {
	m       *installation
	q       port.QueryService
	partner int64
}

// Values a physical row carries until the objects, versions, and trees they name are read.
type (
	pendingRef    int64
	pendingPin    struct{ object, version int64 }
	pendingJSON   int64
	pendingSchema int64
)

func (r *relationalReader) query(ctx context.Context, name string, args ...any) ([][]any, error) {
	res, err := r.q.Query(ctx, name, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if res == nil {
		return nil, nil
	}
	return res.Rows, nil
}

// rowsBy reads a table's rows whose column holds one of the values, each as column name to normalized value.
func (r *relationalReader) rowsBy(ctx context.Context, table, column string, values []int64) ([]map[string]any, error) {
	if len(values) == 0 {
		return nil, nil
	}
	def := r.m.cat.tables[table]
	raw, err := r.query(ctx, qSelectBy(table, column), r.partner, values)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(raw))
	for _, row := range raw {
		if len(row) != len(def.Columns) {
			return nil, fmt.Errorf("%w: %s returned %d columns", ErrDocumentIncomplete, table, len(row))
		}
		m := make(map[string]any, len(row))
		for i, c := range def.Columns {
			v, err := normalize(c, row[i])
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", table, c.Name, err)
			}
			m[c.Name] = v
		}
		out = append(out, m)
	}
	return out, nil
}

// normalize turns a driver value into its logical form: int64, bool, string, json.Number, time.Time, or a date string.
func normalize(c tableColumn, v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	typ := columnType(c)
	bad := fmt.Errorf("%w: %T is not %s", ErrDocumentIncomplete, v, typ)
	switch {
	case typ == "BIGINT" || typ == "INT" || typ == "INTEGER" || typ == "SMALLINT":
		if i, ok := common.AsInt64OK(v); ok {
			return i, nil
		}
		return nil, bad
	case typ == "BOOLEAN":
		if b, ok := v.(bool); ok {
			return b, nil
		}
		return nil, bad
	case typ == "NUMERIC":
		s, ok := v.(string)
		if !ok {
			return nil, bad
		}
		return json.Number(s), nil
	case typ == "TIMESTAMP":
		if t, ok := common.AsTimeOK(v); ok {
			return t.UTC(), nil
		}
		return nil, bad
	case typ == "DATE":
		if t, ok := v.(time.Time); ok {
			return t.Format(dateLayout), nil
		}
		if s, ok := v.(string); ok {
			return parseDate(s)
		}
		return nil, bad
	case strings.HasPrefix(typ, "CHAR("):
		s, ok := v.(string)
		if !ok {
			return nil, bad
		}
		return strings.TrimRight(s, " "), nil
	}
	if s, ok := v.(string); ok {
		return s, nil
	}
	return nil, bad
}

// objectsByID reads enterprise_object rows.
func (r *relationalReader) objectsByID(ctx context.Context, ids []int64) (map[int64]storedObject, error) {
	rows, err := r.rowsBy(ctx, "enterprise_object", "id", ids)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]storedObject, len(rows))
	for _, row := range rows {
		o := objectOf(row)
		out[o.id] = o
	}
	return out, nil
}

func objectOf(row map[string]any) storedObject {
	o := storedObject{id: row["id"].(int64)}
	o.key.NS, _ = row["namespace"].(string)
	o.key.ID, _ = row["identifier"].(string)
	o.code, _ = row["kind"].(string)
	if o.code == externalKindCode {
		o.key.Ext, _ = row["external_kind"].(string)
	}
	if e, ok := row["enterprise_id"].(int64); ok {
		o.enterprise = &e
	}
	return o
}

// objectRows reads objects by name or kind through the dedicated lookups.
func (r *relationalReader) objectRows(ctx context.Context, name string, args ...any) ([]storedObject, error) {
	def := r.m.cat.tables["enterprise_object"]
	raw, err := r.query(ctx, name, append([]any{r.partner}, args...)...)
	if err != nil {
		return nil, err
	}
	out := make([]storedObject, 0, len(raw))
	for _, row := range raw {
		m := map[string]any{}
		for i, c := range def.Columns {
			v, err := normalize(c, row[i])
			if err != nil {
				return nil, err
			}
			m[c.Name] = v
		}
		out = append(out, objectOf(m))
	}
	return out, nil
}

// document returns the stored document named by namespace and identifier; ok is false when none is stored.
func (r *relationalReader) document(ctx context.Context, ns, id string) (storedObject, bool, error) {
	objs, err := r.objectRows(ctx, qObjectByName, ns, id)
	if err != nil {
		return storedObject{}, false, err
	}
	for _, o := range objs {
		if o.code != externalKindCode {
			return o, true, nil
		}
	}
	return storedObject{}, false, nil
}

// versions reads each object's version numbers and labels.
func (r *relationalReader) versions(ctx context.Context, ids []int64) (map[int64]map[int64]string, error) {
	rows, err := r.rowsBy(ctx, "enterprise_object_version", "enterprise_object_id", ids)
	if err != nil {
		return nil, err
	}
	out := map[int64]map[int64]string{}
	for _, row := range rows {
		id, no := row["enterprise_object_id"].(int64), row["version_no"].(int64)
		if out[id] == nil {
			out[id] = map[int64]string{}
		}
		label, _ := row["definition_version"].(string)
		out[id][no] = label
	}
	return out, nil
}

// current sets each object's version to its highest; an object without versions is incomplete.
func (r *relationalReader) current(ctx context.Context, objs []storedObject) error {
	ids := make([]int64, len(objs))
	for i, o := range objs {
		ids[i] = o.id
	}
	versions, err := r.versions(ctx, ids)
	if err != nil {
		return err
	}
	for i := range objs {
		for no := range versions[objs[i].id] {
			objs[i].version = max(objs[i].version, no)
		}
		if objs[i].version == 0 {
			return fmt.Errorf("%w: %s has no version", ErrDocumentIncomplete, objs[i].key)
		}
	}
	return nil
}

// load reads the logical rows of documents of one kind, each at its chosen version.
func (r *relationalReader) load(ctx context.Context, spec *kindSpec, objs []storedObject) ([]*loadedDocument, error) {
	byID := map[int64]*loadedDocument{}
	ids := make([]int64, 0, len(objs))
	out := make([]*loadedDocument, 0, len(objs))
	for _, o := range objs {
		d := &loadedDocument{object: o, rows: lrows{}}
		byID[o.id] = d
		ids = append(ids, o.id)
		out = append(out, d)
	}
	var visitErr error
	physical := map[string][]map[string]any{}
	spec.tables(func(t *tableSpec, l *listSpec) {
		if visitErr != nil {
			return
		}
		tables := []string{t.name}
		if l != nil {
			tables = tables[:0]
			for _, target := range l.targets {
				tables = append(tables, target.table)
			}
		}
		for _, table := range tables {
			if !r.m.tables[table] {
				continue
			}
			def := r.m.cat.tables[table]
			rows, err := r.rowsBy(ctx, table, def.PrimaryKey[1], ids)
			if err != nil {
				visitErr = err
				return
			}
			physical[table] = rows
		}
	})
	if visitErr != nil {
		return nil, visitErr
	}
	for table, rows := range physical {
		io := r.m.io[table]
		for _, row := range rows {
			d := byID[row[io.spec.owner[0]].(int64)]
			if d == nil {
				continue
			}
			if slices.Contains(io.spec.owner, "version_no") && row["version_no"] != d.object.version {
				continue
			}
			lr, name := r.logical(io, row)
			d.rows[name] = append(d.rows[name], lr)
		}
	}
	if err := r.derive(ctx, spec, out, physical); err != nil {
		return nil, err
	}
	if err := r.resolve(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

// logical converts a physical row; the returned name is the pseudo table of a split member.
func (r *relationalReader) logical(io *tableIO, row map[string]any) (lrow, string) {
	if io.split != nil {
		lr := lrow{colSplitRef: pendingRef(row[io.split.col].(int64))}
		for i, c := range io.split.owner {
			lr[c] = row[io.def.PrimaryKey[1+i]]
		}
		lr[io.split.owner[0]] = selfID{}
		return lr, io.split.pseudo
	}
	lr := lrow{}
	for _, c := range io.def.written() {
		v := row[c.Name]
		switch {
		case c.Name == "partner_id" || v == nil:
			continue
		case c.Name == io.spec.owner[0]:
			lr[c.Name] = selfID{}
		case c.Name == "version_no" && slices.Contains(io.spec.owner, "version_no"):
			lr[c.Name] = selfVersion{}
		case io.pins[c.Name] != "":
			lr[c.Name] = pendingPin{object: row[io.pins[c.Name]].(int64), version: v.(int64)}
		case io.refArcs[c.Name] != "":
			lr[io.refArcs[c.Name]] = pendingRef(v.(int64))
		case io.refs[c.Name]:
			lr[c.Name] = pendingRef(v.(int64))
		case io.trees[c.Name] == 'j':
			lr[c.Name] = pendingJSON(v.(int64))
		case io.trees[c.Name] == 's':
			lr[c.Name] = pendingSchema(v.(int64))
		case io.measured && (c.Name == "measured_amount" || c.Name == "measured_quantity"):
			if i, ok := v.(int64); ok {
				v = json.Number(strconv.FormatInt(i, 10))
			}
			lr[colMeasured] = v
		case io.approved && c.Name == "approved_action_class":
			lr[colApproved] = v
		default:
			lr[c.Name] = v
		}
	}
	return lr, io.spec.name
}

// derive finds the identity an agent runtime acts as: for an AgentRuntime its own, and for an ActionRecord whose actor
// is no human, its runtime's.
func (r *relationalReader) derive(ctx context.Context, spec *kindSpec, docs []*loadedDocument, physical map[string][]map[string]any) error {
	definitionOf := map[int64]int64{}
	switch spec.kind {
	case model.KindAgentRuntime:
		for _, row := range physical[spec.root.name] {
			definitionOf[row["id"].(int64)] = row["agent_definition_id"].(int64)
		}
	case model.KindActionRecord:
		runtimeOf := map[int64]int64{}
		var runtimes []int64
		for _, row := range physical[spec.root.name] {
			if row["actor_human_id"] == nil {
				runtimeOf[row["id"].(int64)] = row["runtime_instance_id"].(int64)
				runtimes = append(runtimes, row["runtime_instance_id"].(int64))
			}
		}
		defs, err := r.definitionsOfRuntimes(ctx, runtimes)
		if err != nil {
			return err
		}
		for doc, rt := range runtimeOf {
			if def, ok := defs[rt]; ok {
				definitionOf[doc] = def
			}
		}
	default:
		return nil
	}
	var defIDs []int64
	for _, def := range definitionOf {
		defIDs = append(defIDs, def)
	}
	identities, err := r.identitiesOfDefinitions(ctx, defIDs)
	if err != nil {
		return err
	}
	for _, d := range docs {
		if identity, ok := identities[definitionOf[d.object.id]]; ok {
			d.rows["#agent"] = []lrow{{"agent": pendingRef(identity)}}
		}
	}
	return nil
}

func (r *relationalReader) definitionsOfRuntimes(ctx context.Context, runtimes []int64) (map[int64]int64, error) {
	rows, err := r.rowsBy(ctx, "enterprise_agent_runtime", "id", runtimes)
	if err != nil {
		return nil, err
	}
	out := map[int64]int64{}
	for _, row := range rows {
		out[row["id"].(int64)] = row["agent_definition_id"].(int64)
	}
	return out, nil
}

func (r *relationalReader) identitiesOfDefinitions(ctx context.Context, defs []int64) (map[int64]int64, error) {
	rows, err := r.rowsBy(ctx, "enterprise_agent_definition", "id", defs)
	if err != nil {
		return nil, err
	}
	out := map[int64]int64{}
	for _, row := range rows {
		out[row["id"].(int64)] = row["agent_identity_id"].(int64)
	}
	return out, nil
}

// resolve replaces pending ids with the objects, version labels, and trees they name, and reads currency exponents.
func (r *relationalReader) resolve(ctx context.Context, docs []*loadedDocument) error {
	refs, pins, jsonRoots, schemaRoots := map[int64]bool{}, map[int64]bool{}, map[int64]bool{}, map[int64]bool{}
	currencies := map[string]bool{}
	for _, d := range docs {
		if d.object.enterprise != nil {
			refs[*d.object.enterprise] = true
		}
		for _, rows := range d.rows {
			for _, row := range rows {
				for col, v := range row {
					switch x := v.(type) {
					case pendingRef:
						refs[int64(x)] = true
					case pendingPin:
						refs[x.object], pins[x.object] = true, true
					case pendingJSON:
						jsonRoots[int64(x)] = true
					case pendingSchema:
						schemaRoots[int64(x)] = true
					case string:
						if col == "currency_code" {
							currencies[x] = true
						}
					}
				}
			}
		}
	}
	objects, err := r.objectsByID(ctx, keys(refs))
	if err != nil {
		return err
	}
	labels, err := r.versions(ctx, keys(pins))
	if err != nil {
		return err
	}
	values, err := r.jsonValues(ctx, keys(jsonRoots))
	if err != nil {
		return err
	}
	schemas, err := r.schemas(ctx, keys(schemaRoots))
	if err != nil {
		return err
	}
	exponents, err := r.exponents(ctx, currencies)
	if err != nil {
		return err
	}
	keyOf := func(id int64) (objKey, error) {
		o, ok := objects[id]
		if !ok {
			return objKey{}, fmt.Errorf("%w: object %d", ErrDocumentIncomplete, id)
		}
		return o.key, nil
	}
	kindOf := map[objKey]model.Kind{}
	for _, o := range objects {
		if o.code != externalKindCode {
			kindOf[o.key] = model.Kind(kindCodes.words[o.code])
		}
	}
	for _, d := range docs {
		d.kindOf, d.exponent = kindOf, exponents
		if d.object.enterprise != nil {
			k, err := keyOf(*d.object.enterprise)
			if err != nil {
				return err
			}
			d.enterprise = &k
		}
		for _, rows := range d.rows {
			for _, row := range rows {
				for col, v := range row {
					switch x := v.(type) {
					case pendingRef:
						k, err := keyOf(int64(x))
						if err != nil {
							return err
						}
						row[col] = k
					case pendingPin:
						k, err := keyOf(x.object)
						if err != nil {
							return err
						}
						label, ok := labels[x.object][x.version]
						if !ok || label == "" {
							return fmt.Errorf("%w: %s version %d has no label", ErrDocumentIncomplete, k, x.version)
						}
						row[col] = versionLabel{Object: k, Label: label}
					case pendingJSON:
						row[col] = jsonTree{Value: values[int64(x)]}
					case pendingSchema:
						row[col] = schemaTree{Value: schemas[int64(x)]}
					}
				}
				if cap, ok := row["approved_capability_id"].(objKey); ok {
					row[colApproved] = cap.ID
					delete(row, "approved_capability_id")
				}
			}
		}
		if agent := d.rows["#agent"]; len(agent) == 1 {
			k := agent[0]["agent"].(objKey)
			d.agent = &k
			delete(d.rows, "#agent")
		}
	}
	return nil
}

func keys(m map[int64]bool) []int64 {
	out := make([]int64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (r *relationalReader) exponents(ctx context.Context, codes map[string]bool) (map[string]int64, error) {
	out := map[string]int64{}
	if len(codes) == 0 {
		return out, nil
	}
	list := make([]string, 0, len(codes))
	for c := range codes {
		list = append(list, c)
	}
	sort.Strings(list)
	raw, err := r.query(ctx, qCurrencies, list)
	if err != nil {
		return nil, err
	}
	for _, row := range raw {
		exp, ok := common.AsInt64OK(row[1])
		if !ok {
			return nil, fmt.Errorf("%w: currency exponent %v", ErrDocumentIncomplete, row[1])
		}
		out[strings.TrimRight(common.AsString(row[0]), " ")] = exp
	}
	return out, nil
}

// jsonValues reads json_value trees by root id.
func (r *relationalReader) jsonValues(ctx context.Context, roots []int64) (map[int64]any, error) {
	out := map[int64]any{}
	if len(roots) == 0 {
		return out, nil
	}
	nodes, children := map[int64]*jsonValueRow{}, map[int64][]*jsonValueRow{}
	level, column := roots, "id"
	for len(level) > 0 {
		rows, err := r.rowsBy(ctx, "json_value", column, level)
		if err != nil {
			return nil, err
		}
		level, column = nil, "parent_id"
		for _, row := range rows {
			n := &jsonValueRow{id: row["id"].(int64)}
			if _, seen := nodes[n.id]; seen {
				return nil, fmt.Errorf("%w: json_value %d is reachable twice", ErrDocumentIncomplete, n.id)
			}
			n.valueType, _ = row["value_type"].(string)
			if p, ok := row["parent_id"].(int64); ok {
				n.parent = p
				children[p] = append(children[p], n)
			}
			if s, ok := row["member_name"].(string); ok {
				n.member = &s
			}
			if i, ok := row["item_index"].(int64); ok {
				n.index = &i
			}
			if s, ok := row["scalar_value"].(string); ok {
				n.scalarValue = &s
			}
			nodes[n.id] = n
			level = append(level, n.id)
		}
	}
	for _, id := range roots {
		v, err := buildJSON(id, nodes, children)
		if err != nil {
			return nil, err
		}
		out[id] = v
	}
	return out, nil
}

// schemaTables are json_schema_node's child tables.
var schemaTables = []string{"json_schema_type", "json_schema_required", "json_schema_dependent_required", "json_schema_enum",
	"json_schema_example", "json_schema_vocabulary", "json_schema_annotation"}

// schemas reads json_schema_node trees by root id, each in canonical form.
func (r *relationalReader) schemas(ctx context.Context, roots []int64) (map[int64]any, error) {
	out := map[int64]any{}
	if len(roots) == 0 {
		return out, nil
	}
	nodes := map[int64]*schemaNode{}
	parentOf := map[int64]map[string]any{}
	var all []int64
	level, column := roots, "id"
	for len(level) > 0 {
		rows, err := r.rowsBy(ctx, "json_schema_node", column, level)
		if err != nil {
			return nil, err
		}
		level, column = nil, "parent_id"
		for _, row := range rows {
			id := row["id"].(int64)
			if _, seen := nodes[id]; seen {
				return nil, fmt.Errorf("%w: json_schema_node %d is reachable twice", ErrDocumentIncomplete, id)
			}
			n := &schemaNode{columns: map[string]any{}, annotations: map[string]any{}}
			if b, ok := row["boolean_schema"].(bool); ok {
				n.boolean = &b
			}
			for _, kw := range schemaKeywords {
				if v := row[kw.col]; v != nil {
					n.columns[kw.col] = v
				}
			}
			nodes[id], parentOf[id] = n, row
			all = append(all, id)
			level = append(level, id)
		}
	}
	child := map[string][]map[string]any{}
	valueIDs := map[int64]bool{}
	for _, row := range parentOf {
		for _, c := range []string{"const_value_id", "default_value_id"} {
			if id, ok := row[c].(int64); ok {
				valueIDs[id] = true
			}
		}
	}
	for _, table := range schemaTables {
		rows, err := r.rowsBy(ctx, table, "json_schema_node_id", all)
		if err != nil {
			return nil, err
		}
		child[table] = rows
		for _, row := range rows {
			if id, ok := row["json_value_id"].(int64); ok {
				valueIDs[id] = true
			}
		}
	}
	values, err := r.jsonValues(ctx, keys(valueIDs))
	if err != nil {
		return nil, err
	}
	for id, row := range parentOf {
		n := nodes[id]
		if v, ok := row["const_value_id"].(int64); ok {
			n.constValue = &jsonTree{Value: values[v]}
		}
		if v, ok := row["default_value_id"].(int64); ok {
			n.defaultVal = &jsonTree{Value: values[v]}
		}
		if p, ok := row["parent_id"].(int64); ok {
			parent := nodes[p]
			if parent == nil {
				return nil, fmt.Errorf("%w: json_schema_node %d parent", ErrDocumentIncomplete, id)
			}
			c := schemaChild{node: n}
			c.applicator, _ = row["applicator"].(string)
			if k, ok := row["applicator_key"].(string); ok {
				c.key = &k
			}
			if i, ok := row["applicator_index"].(int64); ok {
				c.index = &i
			}
			parent.children = append(parent.children, c)
		}
	}
	ordered := func(rows []map[string]any) []map[string]any {
		sort.Slice(rows, func(i, j int) bool { return asInt(rows[i]["ordinal"]) < asInt(rows[j]["ordinal"]) })
		return rows
	}
	for _, row := range child["json_schema_type"] {
		n := nodes[row["json_schema_node_id"].(int64)]
		n.types = append(n.types, row["json_type"].(string))
	}
	for _, row := range child["json_schema_required"] {
		n := nodes[row["json_schema_node_id"].(int64)]
		n.required = append(n.required, row["property_name"].(string))
	}
	for _, row := range child["json_schema_dependent_required"] {
		n := nodes[row["json_schema_node_id"].(int64)]
		n.dependent = append(n.dependent, [2]string{row["property_name"].(string), row["required_name"].(string)})
	}
	for _, row := range ordered(child["json_schema_enum"]) {
		n := nodes[row["json_schema_node_id"].(int64)]
		n.enum = append(n.enum, values[row["json_value_id"].(int64)])
	}
	for _, row := range ordered(child["json_schema_example"]) {
		n := nodes[row["json_schema_node_id"].(int64)]
		n.examples = append(n.examples, values[row["json_value_id"].(int64)])
	}
	for _, row := range child["json_schema_vocabulary"] {
		n := nodes[row["json_schema_node_id"].(int64)]
		if n.vocabulary == nil {
			n.vocabulary = map[string]bool{}
		}
		n.vocabulary[row["vocabulary_uri"].(string)] = row["required"].(bool)
	}
	for _, row := range child["json_schema_annotation"] {
		n := nodes[row["json_schema_node_id"].(int64)]
		n.annotations[row["keyword"].(string)] = values[row["json_value_id"].(int64)]
	}
	for _, id := range roots {
		n, ok := nodes[id]
		if !ok {
			return nil, fmt.Errorf("%w: json_schema_node %d", ErrDocumentIncomplete, id)
		}
		out[id] = n.encode()
	}
	return out, nil
}
