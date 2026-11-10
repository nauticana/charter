package keel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/nauticana/keel/common"
	"github.com/nauticana/keel/port"

	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/model"
)

// relationalWriter writes decomposed documents inside one transaction.
type relationalWriter struct {
	relationalReader
	objects map[objKey]storedObject
	planned map[int64]map[string]int64 // version labels this write assigns: object id, label, version number
}

func newRelationalWriter(m *installation, tx port.QueryService, partner int64) *relationalWriter {
	return &relationalWriter{relationalReader: relationalReader{m: m, q: tx, partner: partner},
		objects: map[objKey]storedObject{}, planned: map[int64]map[string]int64{}}
}

// plan is what a write does with one document.
type plan struct {
	d           *docRows
	object      storedObject
	newIdentity bool
	replay      bool
	update      bool
}

// write stores documents: evidence only when new, a definition version only once, and anything else in place.
func (w *relationalWriter) write(ctx context.Context, docs []*docRows, appendOnly bool) error {
	ordered := slices.Clone(docs)
	slices.SortStableFunc(ordered, func(a, b *docRows) int {
		return boolOrder(a.spec.kind != model.KindEnterprise) - boolOrder(b.spec.kind != model.KindEnterprise)
	})
	plans := make([]*plan, 0, len(ordered))
	for _, d := range ordered {
		p, err := w.plan(ctx, d, appendOnly)
		if err != nil {
			return err
		}
		plans = append(plans, p)
	}
	var ops []*rowOp
	for _, p := range plans {
		if p.replay {
			continue
		}
		if err := w.checkDeclared(ctx, p.d); err != nil {
			return err
		}
		if p.update {
			if err := w.deleteContent(ctx, p); err != nil {
				return err
			}
		}
		rowOps, err := w.operations(ctx, p)
		if err != nil {
			return err
		}
		ops = append(ops, rowOps...)
	}
	for _, op := range orderOps(w.m.cat, ops) {
		if err := w.execute(ctx, op); err != nil {
			return err
		}
	}
	for _, p := range plans {
		if p.replay {
			continue
		}
		if err := w.checkAgents(ctx, p); err != nil {
			return err
		}
		if err := w.checkCurrencies(ctx, p.d); err != nil {
			return err
		}
	}
	return nil
}

func boolOrder(b bool) int {
	if b {
		return 1
	}
	return 0
}

// plan registers the document's object and decides between insert, replay, refusal, and an in-place update.
func (w *relationalWriter) plan(ctx context.Context, d *docRows, appendOnly bool) (*plan, error) {
	key := d.key()
	var enterprise *int64
	if d.enterprise != nil {
		e, err := w.object(ctx, *d.enterprise)
		if err != nil {
			return nil, fmt.Errorf("%s enterprise: %w", key, err)
		}
		if e.code != kindCodes.codes[string(model.KindEnterprise)] {
			return nil, fmt.Errorf("%w: %s enterpriseId names %s", ErrUnrepresentable, key, e.key)
		}
		enterprise = &e.id
	}
	stored, existed, err := w.document(ctx, key.NS, key.ID)
	if err != nil {
		return nil, err
	}
	p := &plan{d: d}
	if !existed {
		id, err := w.insertObject(ctx, map[string]any{"namespace": key.NS, "identifier": key.ID, "kind": d.spec.code, "enterprise_id": enterprise})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		p.object = storedObject{id: id, key: key, code: d.spec.code, enterprise: enterprise, version: 1}
		p.newIdentity = true
		w.objects[key] = p.object
		w.assign(p)
		return p, nil
	}
	switch {
	case appendOnly:
		return nil, fmt.Errorf("%w: %s", evidence.ErrDuplicate, key)
	case stored.code != d.spec.code:
		return nil, fmt.Errorf("%w: %s is %s, not %s", ErrKindChanged, key, kindCodes.words[stored.code], d.spec.kind)
	case (stored.enterprise == nil) != (enterprise == nil) || (enterprise != nil && *stored.enterprise != *enterprise):
		return nil, fmt.Errorf("%w: %s", ErrEnterpriseChanged, key)
	}
	w.objects[key] = stored
	p.object = stored
	versions, err := w.versions(ctx, []int64{stored.id})
	if err != nil {
		return nil, err
	}
	current := stored
	for no := range versions[stored.id] {
		current.version = max(current.version, no)
	}
	if current.version == 0 {
		return nil, fmt.Errorf("%w: %s has no version", ErrDocumentIncomplete, key)
	}
	if d.spec.versioned() {
		labelled, found := current, false
		for no, label := range versions[stored.id] {
			if label == d.label {
				labelled.version, found = no, true
			}
		}
		prior, err := w.loadOne(ctx, d.spec, labelled)
		if err != nil {
			return nil, err
		}
		if !sameRows(lrows{"": prior.rows[d.spec.identity.name]}, lrows{"": d.rows[d.spec.identity.name]}) {
			return nil, fmt.Errorf("%w: %s %s", ErrFixedChanged, d.spec.kind, key)
		}
		if found {
			if !sameRows(prior.rows, d.rows) {
				return nil, fmt.Errorf("%w: %s %s version %q is stored with other content", ErrPublishedVersion, d.spec.kind, key, d.label)
			}
			p.replay = true
			return p, nil
		}
		p.object.version = current.version + 1
		w.assign(p)
		return p, nil
	}
	p.object.version = current.version
	prior, err := w.loadOne(ctx, d.spec, current)
	if err != nil {
		return nil, err
	}
	if sameRows(prior.rows, d.rows) {
		p.replay = true
		return p, nil
	}
	if d.spec.once {
		return nil, fmt.Errorf("%w: %s %s", ErrRecordWritten, d.spec.kind, key)
	}
	for _, f := range d.spec.root.fields {
		if !f.fixed {
			continue
		}
		for _, c := range f.columns() {
			if canonicalValue(prior.rows[d.spec.root.name][0][c]) != canonicalValue(d.rows[d.spec.root.name][0][c]) {
				return nil, fmt.Errorf("%w: %s %s %s", ErrFixedChanged, d.spec.kind, key, f.path)
			}
		}
	}
	p.update = true
	return p, nil
}

// assign records the version a write gives a definition label, so pins in the same write resolve to it.
func (w *relationalWriter) assign(p *plan) {
	if p.d.label == "" {
		return
	}
	if w.planned[p.object.id] == nil {
		w.planned[p.object.id] = map[string]int64{}
	}
	w.planned[p.object.id][p.d.label] = p.object.version
}

func (w *relationalWriter) loadOne(ctx context.Context, spec *kindSpec, o storedObject) (*loadedDocument, error) {
	docs, err := w.load(ctx, spec, []storedObject{o})
	if err != nil {
		return nil, err
	}
	return docs[0], nil
}

// object resolves a reference; an object outside the document set is registered when first referenced.
func (w *relationalWriter) object(ctx context.Context, key objKey) (storedObject, error) {
	if o, ok := w.objects[key]; ok {
		return o, nil
	}
	objs, err := w.objectRows(ctx, qObjectByName, key.NS, key.ID)
	if err != nil {
		return storedObject{}, err
	}
	for _, o := range objs {
		if (key.Ext == "") == (o.code != externalKindCode) && o.key == key {
			w.objects[key] = o
			return o, nil
		}
	}
	if key.Ext == "" {
		return storedObject{}, fmt.Errorf("%w: %s", ErrUnresolvedRef, key)
	}
	id, err := w.insertObject(ctx, map[string]any{"namespace": key.NS, "identifier": key.ID, "kind": externalKindCode, "external_kind": key.Ext})
	if err != nil {
		return storedObject{}, fmt.Errorf("%s: %w", key, err)
	}
	o := storedObject{id: id, key: key, code: externalKindCode}
	w.objects[key] = o
	return o, nil
}

// lookup resolves a reference that may name nothing stored.
func (w *relationalWriter) lookup(ctx context.Context, key objKey) (storedObject, bool, error) {
	o, err := w.object(ctx, key)
	if errors.Is(err, ErrUnresolvedRef) {
		return storedObject{}, false, nil
	}
	return o, err == nil, err
}

func (w *relationalWriter) insertObject(ctx context.Context, row map[string]any) (int64, error) {
	return w.insert(ctx, "enterprise_object", row)
}

// insert writes one physical row and returns the id its sequence assigned, if any.
func (w *relationalWriter) insert(ctx context.Context, table string, row map[string]any) (int64, error) {
	def := w.m.cat.tables[table]
	args := []any{}
	for _, c := range def.written() {
		if def.Sequence != nil && c.Name == def.Sequence.Column {
			continue
		}
		v := row[c.Name]
		if c.Name == "partner_id" {
			v = w.partner
		}
		args = append(args, bind(c, v))
	}
	raw, err := w.query(ctx, qInsert(table), args...)
	if err != nil {
		return 0, err
	}
	if def.Sequence == nil {
		return 0, nil
	}
	if len(raw) != 1 || len(raw[0]) != 1 {
		return 0, fmt.Errorf("%w: %s insert returned no id", ErrDocumentIncomplete, table)
	}
	id, ok := common.AsInt64OK(raw[0][0])
	if !ok {
		return 0, fmt.Errorf("%w: %s insert returned id %v", ErrDocumentIncomplete, table, raw[0][0])
	}
	return id, nil
}

// bind converts a physical value to its driver argument: NUMERIC as text and DATE as a midnight UTC time.
func bind(c tableColumn, v any) any {
	switch x := v.(type) {
	case json.Number:
		return string(x)
	case string:
		if columnType(c) == "DATE" {
			t, _ := time.Parse(dateLayout, x)
			return t
		}
	case *int64:
		if x != nil {
			return *x
		}
		return nil
	case *string:
		if x != nil {
			return *x
		}
		return nil
	case *bool:
		if x != nil {
			return *x
		}
		return nil
	}
	return v
}

func (w *relationalWriter) checkDeclared(ctx context.Context, d *docRows) error {
	for _, ref := range d.declared {
		o, err := w.object(ctx, ref.key)
		if err != nil {
			return fmt.Errorf("%s: %w", d.key(), err)
		}
		if want := kindCodes.codes[string(ref.kind)]; o.code != want {
			return fmt.Errorf("%w: %s names %s as %s", ErrUnrepresentable, d.key(), ref.key, ref.kind)
		}
	}
	return nil
}

// deleteContent removes a mutable document's child rows, children first, and the JSON trees they owned.
func (w *relationalWriter) deleteContent(ctx context.Context, p *plan) error {
	var tables []string
	p.d.spec.tables(func(t *tableSpec, l *listSpec) {
		if t == p.d.spec.envelope || t == p.d.spec.root {
			return
		}
		if l != nil {
			for _, target := range l.targets {
				tables = append(tables, target.table)
			}
			return
		}
		tables = append(tables, t.name)
	})
	var roots []int64
	for i := len(tables) - 1; i >= 0; i-- {
		table := tables[i]
		if !w.m.tables[table] {
			continue
		}
		io := w.m.io[table]
		col := io.def.PrimaryKey[1]
		if len(io.trees) > 0 {
			rows, err := w.rowsBy(ctx, table, col, []int64{p.object.id})
			if err != nil {
				return err
			}
			for _, row := range rows {
				for c, kind := range io.trees {
					if kind != 'j' {
						return fmt.Errorf("%w: a schema tree in mutable %s", ErrUnrepresentable, table)
					}
					if id, ok := row[c].(int64); ok {
						roots = append(roots, id)
					}
				}
			}
		}
		if _, err := w.query(ctx, qDeleteBy(table, col), w.partner, []int64{p.object.id}); err != nil {
			return err
		}
	}
	return w.deleteJSON(ctx, roots)
}

// deleteJSON removes trees by root id; the first level read is the roots themselves.
func (w *relationalWriter) deleteJSON(ctx context.Context, roots []int64) error {
	var all []int64
	level, column := roots, "id"
	for len(level) > 0 {
		rows, err := w.rowsBy(ctx, "json_value", column, level)
		if err != nil {
			return err
		}
		level, column = nil, "parent_id"
		for _, row := range rows {
			level = append(level, row["id"].(int64))
		}
		all = append(all, level...)
	}
	if len(all) == 0 {
		return nil
	}
	_, err := w.query(ctx, qDeleteBy("json_value", "id"), w.partner, all)
	return err
}

// rowOp is one physical row to insert, or to update by key.
type rowOp struct {
	table  string
	row    map[string]any
	update bool
}

// operations converts a document's logical rows to physical rows, writing its JSON trees first.
func (w *relationalWriter) operations(ctx context.Context, p *plan) ([]*rowOp, error) {
	var ops []*rowOp
	var visitErr error
	spec := p.d.spec
	spec.tables(func(t *tableSpec, l *listSpec) {
		if visitErr != nil {
			return
		}
		if t == spec.identity && !p.newIdentity {
			return
		}
		for _, lr := range p.d.rows[t.name] {
			table, row, err := w.physical(ctx, p, t, l, lr)
			if err != nil {
				visitErr = fmt.Errorf("%s %s: %w", p.d.key(), t.name, err)
				return
			}
			update := p.update && (t == spec.envelope || t == spec.root)
			ops = append(ops, &rowOp{table: table, row: row, update: update})
		}
	})
	return ops, visitErr
}

func (w *relationalWriter) physical(ctx context.Context, p *plan, t *tableSpec, l *listSpec, lr lrow) (string, map[string]any, error) {
	resolve := func(v any) (any, error) {
		switch x := v.(type) {
		case selfID:
			return p.object.id, nil
		case selfVersion:
			return p.object.version, nil
		case objKey:
			o, err := w.object(ctx, x)
			return o.id, err
		case versionLabel:
			return w.versionOf(ctx, x)
		case jsonTree:
			return w.insertJSON(ctx, x.Value, nil, nil, nil)
		case schemaTree:
			node, err := decodeSchema(x.Value)
			if err != nil {
				return nil, err
			}
			return w.insertSchema(ctx, node, nil, schemaChild{})
		}
		return v, nil
	}
	row := map[string]any{}
	if l != nil {
		key := lr[colSplitRef].(objKey)
		o, err := w.object(ctx, key)
		if err != nil {
			return "", nil, err
		}
		for _, target := range l.targets {
			if kindCodes.codes[string(target.kind)] == o.code {
				row[t.owner[0]] = p.object.id
				row[target.col] = o.id
				return target.table, row, nil
			}
		}
		return "", nil, fmt.Errorf("%w: %s is %s, which %s does not admit", ErrUnrepresentable, key, kindCodes.words[o.code], l.path)
	}
	for col, v := range lr {
		if strings.HasPrefix(col, "#") {
			continue
		}
		resolved, err := resolve(v)
		if err != nil {
			return "", nil, fmt.Errorf("%s: %w", col, err)
		}
		row[col] = resolved
	}
	for _, f := range t.fields {
		switch f.kind {
		case fRefArc:
			key, ok := lr["#"+f.path].(objKey)
			if !ok {
				continue
			}
			o, err := w.object(ctx, key)
			if err != nil {
				return "", nil, err
			}
			placed := false
			for _, a := range f.arcs {
				if kindCodes.codes[string(a.kind)] == o.code {
					row[a.col], placed = o.id, true
				}
			}
			if !placed {
				return "", nil, fmt.Errorf("%w: %s is %s, which %s does not admit", ErrUnrepresentable, key, kindCodes.words[o.code], f.path)
			}
		case fApproved:
			action, ok := lr[colApproved].(string)
			if !ok {
				continue
			}
			o, found, err := w.lookup(ctx, objKey{NS: p.d.namespace, ID: action})
			if err != nil {
				return "", nil, err
			}
			if found && o.code == kindCodes.codes[string(model.KindCapabilityContract)] {
				row["approved_capability_id"] = o.id
			} else {
				row["approved_action_class"] = action
			}
		case fMeasured:
			value, ok := lr[colMeasured].(json.Number)
			if !ok {
				continue
			}
			currency, err := w.measureCurrency(ctx, row["outcome_measure_id"])
			if err != nil {
				return "", nil, err
			}
			if currency {
				i, err := parseInt(value)
				if err != nil {
					return "", nil, fmt.Errorf("%w: a measure with a currency is measured in integer minor units", err)
				}
				row["measured_amount"] = i
			} else {
				row["measured_quantity"] = value
			}
		}
	}
	return t.name, row, nil
}

// versionOf is the version number a label names: one this write assigns, or one already stored.
func (w *relationalWriter) versionOf(ctx context.Context, pin versionLabel) (int64, error) {
	o, err := w.object(ctx, pin.Object)
	if err != nil {
		return 0, err
	}
	if no, ok := w.planned[o.id][pin.Label]; ok {
		return no, nil
	}
	versions, err := w.versions(ctx, []int64{o.id})
	if err != nil {
		return 0, err
	}
	for no, label := range versions[o.id] {
		if label == pin.Label {
			return no, nil
		}
	}
	return 0, fmt.Errorf("%w: %s %q", ErrUnknownVersion, pin.Object, pin.Label)
}

func (w *relationalWriter) measureCurrency(ctx context.Context, measure any) (bool, error) {
	id, ok := measure.(int64)
	if !ok {
		return false, fmt.Errorf("%w: a measurement names no measure", ErrUnrepresentable)
	}
	rows, err := w.rowsBy(ctx, "outcome_measure", "id", []int64{id})
	if err != nil {
		return false, err
	}
	if len(rows) != 1 {
		return false, fmt.Errorf("%w: outcome measure %d", ErrUnresolvedRef, id)
	}
	return rows[0]["currency_code"] != nil, nil
}

// insertJSON writes a value as json_value nodes, parents first, and returns the root id.
func (w *relationalWriter) insertJSON(ctx context.Context, v any, parent *int64, member *string, index *int64) (int64, error) {
	valueType, scalar, err := jsonType(v)
	if err != nil {
		return 0, err
	}
	id, err := w.insert(ctx, "json_value", map[string]any{"parent_id": parent, "member_name": member, "item_index": index,
		"value_type": valueType, "scalar_value": scalar})
	if err != nil {
		return 0, err
	}
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			name := k
			if _, err := w.insertJSON(ctx, e, &id, &name, nil); err != nil {
				return 0, err
			}
		}
	case []any:
		for i, e := range x {
			at := int64(i)
			if _, err := w.insertJSON(ctx, e, &id, nil, &at); err != nil {
				return 0, err
			}
		}
	}
	return id, nil
}

// insertSchema writes a schema node, its keyword tables, and its subschemas, and returns the node id.
func (w *relationalWriter) insertSchema(ctx context.Context, n *schemaNode, parent *int64, at schemaChild) (int64, error) {
	row := map[string]any{"parent_id": parent, "applicator_key": at.key, "applicator_index": at.index, "boolean_schema": n.boolean}
	if at.applicator != "" {
		row["applicator"] = at.applicator
	}
	for col, v := range n.columns {
		row[col] = v
	}
	for col, tree := range map[string]*jsonTree{"const_value_id": n.constValue, "default_value_id": n.defaultVal} {
		if tree == nil {
			continue
		}
		id, err := w.insertJSON(ctx, tree.Value, nil, nil, nil)
		if err != nil {
			return 0, err
		}
		row[col] = id
	}
	id, err := w.insert(ctx, "json_schema_node", row)
	if err != nil {
		return 0, err
	}
	node := func(cols map[string]any) map[string]any { cols["json_schema_node_id"] = id; return cols }
	var rows []struct {
		table string
		row   map[string]any
	}
	add := func(table string, r map[string]any) {
		rows = append(rows, struct {
			table string
			row   map[string]any
		}{table, node(r)})
	}
	for _, t := range n.types {
		add("json_schema_type", map[string]any{"json_type": t})
	}
	for _, name := range n.required {
		add("json_schema_required", map[string]any{"property_name": name})
	}
	for _, pair := range n.dependent {
		add("json_schema_dependent_required", map[string]any{"property_name": pair[0], "required_name": pair[1]})
	}
	for uri, req := range n.vocabulary {
		add("json_schema_vocabulary", map[string]any{"vocabulary_uri": uri, "required": req})
	}
	for table, list := range map[string][]any{"json_schema_enum": n.enum, "json_schema_example": n.examples} {
		for i, v := range list {
			value, err := w.insertJSON(ctx, v, nil, nil, nil)
			if err != nil {
				return 0, err
			}
			add(table, map[string]any{"ordinal": int64(i + 1), "json_value_id": value})
		}
	}
	for keyword, v := range n.annotations {
		value, err := w.insertJSON(ctx, v, nil, nil, nil)
		if err != nil {
			return 0, err
		}
		add("json_schema_annotation", map[string]any{"keyword": keyword, "json_value_id": value})
	}
	for _, r := range rows {
		if _, err := w.insert(ctx, r.table, r.row); err != nil {
			return 0, err
		}
	}
	for _, c := range n.children {
		if _, err := w.insertSchema(ctx, c.node, &id, c); err != nil {
			return 0, err
		}
	}
	return id, nil
}

// orderOps sorts row operations so every row follows the rows of this write its foreign keys reference.
func orderOps(cat *tableCatalog, ops []*rowOp) []*rowOp {
	index := map[string]int{}
	for i, op := range ops {
		if op.update {
			continue
		}
		def := cat.tables[op.table]
		index[op.table+"\x00"+physicalKey(def.PrimaryKey, op.row)] = i
	}
	deps := make([][]int, len(ops))
	for i, op := range ops {
		def := cat.tables[op.table]
		for _, fk := range def.ForeignKeys {
			parent := cat.tables[fk.References.Table]
			if parent == nil {
				continue
			}
			values := map[string]any{}
			complete := true
			for j, c := range fk.Columns {
				v := op.row[c]
				if c == "partner_id" {
					v = int64(0)
				}
				if v == nil {
					complete = false
					break
				}
				values[fk.References.Columns[j]] = v
			}
			if !complete {
				continue
			}
			if j, ok := index[fk.References.Table+"\x00"+physicalKey(parent.PrimaryKey, values)]; ok && j != i {
				deps[i] = append(deps[i], j)
			}
		}
	}
	out := make([]*rowOp, 0, len(ops))
	state := make([]uint8, len(ops))
	var visit func(i int)
	visit = func(i int) {
		if state[i] != 0 {
			return
		}
		state[i] = 1
		for _, j := range deps[i] {
			visit(j)
		}
		state[i] = 2
		out = append(out, ops[i])
	}
	for i := range ops {
		visit(i)
	}
	return out
}

func physicalKey(cols []string, row map[string]any) string {
	values := make([]any, 0, len(cols))
	for _, c := range cols {
		v := row[c]
		if c == "partner_id" {
			v = int64(0)
		}
		values = append(values, v)
	}
	return rowKey(values...)
}

func (w *relationalWriter) execute(ctx context.Context, op *rowOp) error {
	if !op.update {
		_, err := w.insert(ctx, op.table, op.row)
		if err != nil {
			return fmt.Errorf("%s: %w", op.table, err)
		}
		return nil
	}
	def := w.m.cat.tables[op.table]
	var args []any
	for _, c := range def.updatedColumns() {
		args = append(args, bind(c, op.row[c.Name]))
	}
	for _, k := range def.PrimaryKey {
		v := op.row[k]
		if k == "partner_id" {
			v = w.partner
		}
		args = append(args, v)
	}
	if _, err := w.query(ctx, qUpdate(op.table), args...); err != nil {
		return fmt.Errorf("%s: %w", op.table, err)
	}
	return nil
}

// checkAgents confirms each agent a document names is the identity of the runtime it acts through.
func (w *relationalWriter) checkAgents(ctx context.Context, p *plan) error {
	for _, check := range p.d.agents {
		runtime := p.object.id
		if !check.self {
			o, err := w.object(ctx, check.runtime)
			if err != nil {
				return err
			}
			runtime = o.id
		}
		agent, err := w.object(ctx, check.agent)
		if err != nil {
			return err
		}
		defs, err := w.definitionsOfRuntimes(ctx, []int64{runtime})
		if err != nil {
			return err
		}
		identities, err := w.identitiesOfDefinitions(ctx, []int64{defs[runtime]})
		if err != nil {
			return err
		}
		if identity, ok := identities[defs[runtime]]; !ok || identity != agent.id {
			return fmt.Errorf("%w: %s names agent %s", ErrRuntimeActor, p.d.key(), check.agent)
		}
	}
	return nil
}

// checkCurrencies confirms each stated currency exponent is keel's.
func (w *relationalWriter) checkCurrencies(ctx context.Context, d *docRows) error {
	if len(d.currencies) == 0 {
		return nil
	}
	codes := map[string]bool{}
	for c := range d.currencies {
		codes[c] = true
	}
	known, err := w.exponents(ctx, codes)
	if err != nil {
		return err
	}
	for c, stated := range d.currencies {
		if exp, ok := known[c]; !ok || exp != stated {
			return fmt.Errorf("%w: %s %s states %d", ErrCurrencyExponent, d.key(), c, stated)
		}
	}
	return nil
}
