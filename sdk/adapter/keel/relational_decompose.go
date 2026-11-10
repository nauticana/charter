package keel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/nauticana/charter/sdk/model"
)

// docRows is a document decomposed into logical rows, with the facts the store checks when it writes them.
type docRows struct {
	spec       *kindSpec
	namespace  string
	id         string
	enterprise *objKey
	label      string
	rows       lrows
	declared   []declaredRef
	currencies map[string]int64
	agents     []agentCheck
}

// declaredRef is an objectRef whose stated kind the store compares with the stored target's.
type declaredRef struct {
	key  objKey
	kind model.Kind
}

// agentCheck states that an agent is the identity of a runtime: the document itself when self is set.
type agentCheck struct {
	self    bool
	runtime objKey
	agent   objKey
}

const (
	pseudoActorAgent   = "#actor_agent"
	pseudoRuntimeAgent = "#runtime_agent"
)

func (d *docRows) key() objKey { return objKey{NS: d.namespace, ID: d.id} }

// topOwner is the owner key of a document's top tables: its id, and its version for a version-keyed table.
func topOwner(t *tableSpec) []any {
	if len(t.owner) == 1 {
		return []any{selfID{}}
	}
	return []any{selfID{}, selfVersion{}}
}

// childOwner is the owner key of a list's rows: the document version for a shared table, else the parent's key.
func childOwner(l *listSpec, parentPK []any) []any {
	if l.table.owner[0] == "enterprise_object_id" {
		return []any{selfID{}, selfVersion{}}
	}
	return parentPK
}

// pk is the key a row's children reference: its owner key with its natural key, or with its ordinal.
func (t *tableSpec) pk(row lrow) []any {
	out := make([]any, 0, len(t.owner)+len(t.key)+1)
	for _, c := range t.owner {
		out = append(out, row[c])
	}
	switch {
	case len(t.key) > 0:
		for _, c := range t.key {
			out = append(out, row[c])
		}
	case t.order != "":
		out = append(out, row[t.order])
	}
	return out
}

func documentValue(d *model.Document) (map[string]any, error) {
	v := d.Value
	if v == nil {
		dec := json.NewDecoder(bytes.NewReader(d.Raw))
		dec.UseNumber()
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
	}
	m, ok := cloneJSON(v).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: document is not an object", ErrUnrepresentable)
	}
	return m, nil
}

// decompose maps a document to logical rows; a property no column holds is an error, so nothing is dropped.
func decompose(d *model.Document) (*docRows, error) {
	spec, ok := kindSpecs[d.Kind]
	if !ok {
		return nil, fmt.Errorf("%w: kind %s", ErrUnmappedProperty, d.Kind)
	}
	value, err := documentValue(d)
	if err != nil {
		return nil, err
	}
	out := &docRows{spec: spec, namespace: d.Namespace, id: d.ID, rows: lrows{}, currencies: map[string]int64{}}
	for property, want := range map[string]string{"namespace": d.Namespace, "id": d.ID, "kind": string(d.Kind)} {
		if v, _ := takePath(value, property); v != want {
			return nil, fmt.Errorf("%w: %s %v disagrees with the envelope", ErrUnrepresentable, property, v)
		}
	}
	if v, ok := takePath(value, "enterpriseId"); ok {
		key, err := parseIDRef(v, d.Namespace)
		if err != nil {
			return nil, err
		}
		out.enterprise = &key
	}
	dc := decomposer{out: out}
	for _, t := range []*tableSpec{spec.envelope, spec.identity, spec.root} {
		if t == nil {
			continue
		}
		if err := dc.table(t, value, topOwner(t), 0); err != nil {
			return nil, fmt.Errorf("%s %s:%s: %w", d.Kind, d.Namespace, d.ID, err)
		}
	}
	if left := leftover(value, ""); len(left) > 0 {
		return nil, fmt.Errorf("%w: %s %s:%s %v", ErrUnmappedProperty, d.Kind, d.Namespace, d.ID, left)
	}
	if spec.versioned() {
		label, _ := out.rows[spec.envelope.name][0]["definition_version"].(string)
		if label == "" {
			return nil, fmt.Errorf("%w: %s %s:%s declares no %s", ErrUnrepresentable, d.Kind, d.Namespace, d.ID, spec.label)
		}
		out.label = label
	}
	root := out.rows[spec.root.name][0]
	if agent, ok := root[pseudoActorAgent].(objKey); ok {
		runtime, _ := root["runtime_instance_id"].(objKey)
		out.agents = append(out.agents, agentCheck{runtime: runtime, agent: agent})
		delete(root, pseudoActorAgent)
	}
	if agent, ok := root[pseudoRuntimeAgent].(objKey); ok {
		out.agents = append(out.agents, agentCheck{self: true, agent: agent})
		delete(root, pseudoRuntimeAgent)
	}
	if cols := spec.actorRuntime; cols[0] != "" {
		agent, agentOK := root[cols[0]].(objKey)
		runtime, runtimeOK := root[cols[1]].(objKey)
		if agentOK && runtimeOK {
			out.agents = append(out.agents, agentCheck{runtime: runtime, agent: agent})
		}
	}
	return out, nil
}

type decomposer struct{ out *docRows }

func (dc decomposer) table(t *tableSpec, item map[string]any, owner []any, ordinal int64) error {
	row := lrow{}
	for i, c := range t.owner {
		row[c] = owner[i]
	}
	if t.order != "" {
		row[t.order] = ordinal
	}
	for _, f := range t.fields {
		if err := dc.field(f, item, row); err != nil {
			return fmt.Errorf("%s: %w", f.path, err)
		}
	}
	dc.out.rows[t.name] = append(dc.out.rows[t.name], row)
	pk := t.pk(row)
	for i := range t.lists {
		if err := dc.list(&t.lists[i], item, pk); err != nil {
			return fmt.Errorf("%s: %w", t.lists[i].path, err)
		}
	}
	return nil
}

func (dc decomposer) list(l *listSpec, item map[string]any, parentPK []any) error {
	v, ok := takePath(item, l.path)
	if !ok {
		return nil
	}
	owner := childOwner(l, parentPK)
	if len(owner) != len(l.table.owner) {
		return fmt.Errorf("%w: %s owner key does not match its parent", ErrUnrepresentable, l.table.name)
	}
	newRow := func() lrow {
		row := lrow{}
		for i, c := range l.table.owner {
			row[c] = owner[i]
		}
		return row
	}
	switch l.shape {
	case lOne:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%w: not an object", ErrUnrepresentable)
		}
		if err := dc.table(l.table, m, owner, 0); err != nil {
			return err
		}
		return noneLeft(m)
	case lMap:
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%w: not an object", ErrUnrepresentable)
		}
		names := make([]string, 0, len(m))
		for k := range m {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, k := range names {
			row := newRow()
			row[l.mapKey] = k
			if err := dc.value(l.table.fields[0], m[k], row); err != nil {
				return fmt.Errorf("%s: %w", k, err)
			}
			dc.out.rows[l.table.name] = append(dc.out.rows[l.table.name], row)
		}
		return nil
	}
	list, ok := v.([]any)
	if !ok {
		return fmt.Errorf("%w: not an array", ErrUnrepresentable)
	}
	for i, e := range list {
		ordinal := int64(i + 1)
		switch {
		case l.shape == lSplit:
			key, err := parseIDRef(e, dc.out.namespace)
			if err != nil {
				return err
			}
			row := newRow()
			row[colSplitRef] = key
			dc.out.rows[l.table.name] = append(dc.out.rows[l.table.name], row)
		case l.scalar:
			row := newRow()
			if l.table.order != "" {
				row[l.table.order] = ordinal
			}
			if err := dc.value(l.table.fields[0], e, row); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
			dc.out.rows[l.table.name] = append(dc.out.rows[l.table.name], row)
		default:
			if s, isString := e.(string); isString && l.table.prose != "" {
				row := newRow()
				row[l.table.order] = ordinal
				row[l.table.prose] = s
				dc.out.rows[l.table.name] = append(dc.out.rows[l.table.name], row)
				continue
			}
			m, ok := e.(map[string]any)
			if !ok {
				return fmt.Errorf("%w: [%d] is not an object", ErrUnrepresentable, i)
			}
			if err := dc.table(l.table, m, owner, ordinal); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
			if err := noneLeft(m); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
	}
	return nil
}

func noneLeft(m map[string]any) error {
	if left := leftover(m, ""); len(left) > 0 {
		return fmt.Errorf("%w: %v", ErrUnmappedProperty, left)
	}
	return nil
}

func (dc decomposer) field(f field, item map[string]any, row lrow) error {
	if f.kind == fPin {
		id, idOK := takePath(item, f.path)
		label, labelOK := takePath(item, f.label)
		if !idOK && !labelOK {
			return nil
		}
		key, err := parseIDRef(id, dc.out.namespace)
		if err != nil {
			return err
		}
		s, ok := label.(string)
		if !labelOK || !ok || s == "" {
			return fmt.Errorf("%w: %s names no version", ErrUnrepresentable, f.path)
		}
		row[f.col], row[f.verCol] = key, versionLabel{Object: key, Label: s}
		return nil
	}
	v, ok := takePath(item, f.path)
	if !ok {
		if f.kind == fFlag {
			row[f.col] = false
		}
		return nil
	}
	return dc.value(f, v, row)
}

func (dc decomposer) value(f field, v any, row lrow) error {
	ns := dc.out.namespace
	switch f.kind {
	case fText:
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%w: %v is not a string", ErrUnrepresentable, v)
		}
		row[f.col] = s
	case fInt:
		i, err := parseInt(v)
		if err != nil {
			return err
		}
		row[f.col] = i
	case fNumber:
		n, err := parseNumber(v)
		if err != nil {
			return err
		}
		row[f.col] = n
	case fBool, fFlag:
		b, ok := v.(bool)
		if !ok {
			return fmt.Errorf("%w: %v is not a boolean", ErrUnrepresentable, v)
		}
		row[f.col] = b
	case fTime:
		t, err := parseTime(v)
		if err != nil {
			return err
		}
		row[f.col] = t
	case fDate:
		s, err := parseDate(v)
		if err != nil {
			return err
		}
		row[f.col] = s
	case fCode:
		s, _ := v.(string)
		c, err := f.codes.code(s)
		if err != nil {
			return err
		}
		row[f.col] = c
	case fCodeOrText:
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%w: %v is not a string", ErrUnrepresentable, v)
		}
		if c, err := f.codes.code(s); err == nil {
			row[f.cols[0]] = c
		} else {
			row[f.cols[1]] = s
		}
	case fRef:
		key, err := parseIDRef(v, ns)
		if err != nil {
			return err
		}
		row[f.col] = key
	case fObject:
		key, kind, err := parseObjectRef(v, ns)
		if err != nil {
			return err
		}
		if key.Ext == "" {
			dc.out.declared = append(dc.out.declared, declaredRef{key: key, kind: kind})
		}
		row[f.col] = key
	case fArc, fActor:
		key, kind, err := parseObjectRef(v, ns)
		if err != nil {
			return err
		}
		if f.kind == fActor {
			switch kind {
			case model.KindHumanIdentity:
				row[f.col] = key
			case model.KindAgentIdentity:
				row[pseudoActorAgent] = key
			default:
				return fmt.Errorf("%w: actor kind %s", ErrUnrepresentable, kind)
			}
			if key.Ext != "" {
				return fmt.Errorf("%w: an actor is a Charter identity", ErrUnrepresentable)
			}
			return nil
		}
		for _, a := range f.arcs {
			if a.kind == kind && a.external == (key.Ext != "") {
				row[a.col] = key
				return nil
			}
		}
		return fmt.Errorf("%w: kind %s is not admitted", ErrUnrepresentable, kind)
	case fRefArc:
		key, err := parseIDRef(v, ns)
		if err != nil {
			return err
		}
		row["#"+f.path] = key
	case fRuntimeAgent:
		key, err := parseIDRef(v, ns)
		if err != nil {
			return err
		}
		row[pseudoRuntimeAgent] = key
	case fJSON:
		row[f.col] = jsonTree{Value: v}
	case fSchema:
		node, err := decodeSchema(v)
		if err != nil {
			return err
		}
		row[f.col] = schemaTree{Value: node.encode()}
	case fConst:
		if !reflect.DeepEqual(v, f.value) {
			return fmt.Errorf("%w: must be %v", ErrUnrepresentable, f.value)
		}
	case fApproved:
		s, ok := v.(string)
		if !ok || s == "" {
			return fmt.Errorf("%w: %v is not an action", ErrUnrepresentable, v)
		}
		row[colApproved] = s
	case fMeasured:
		n, err := parseNumber(v)
		if err != nil {
			return err
		}
		row[colMeasured] = n
	case fMoney, fLimitValue:
		return dc.amount(f, v, row)
	case fExponent:
		i, err := parseInt(v)
		if err != nil {
			return err
		}
		c, ok := row["currency_code"].(string)
		if !ok {
			return fmt.Errorf("%w: currencyExponent without a currency", ErrUnrepresentable)
		}
		if prior, seen := dc.out.currencies[c]; seen && prior != i {
			return fmt.Errorf("%w: %s stated with exponents %d and %d", ErrCurrencyExponent, c, prior, i)
		}
		dc.out.currencies[c] = i
	default:
		return fmt.Errorf("%w: field kind %d", ErrUnrepresentable, f.kind)
	}
	return nil
}

// amount stores a value in integer minor units when its item has a currency, otherwise as a quantity, text, or flag.
func (dc decomposer) amount(f field, v any, row lrow) error {
	_, hasCurrency := row["currency_code"].(string)
	switch x := v.(type) {
	case json.Number:
		if hasCurrency {
			i, err := parseInt(x)
			if err != nil {
				return fmt.Errorf("%w: a currency amount is integer minor units", err)
			}
			row[f.cols[0]] = i
			return nil
		}
		n, err := parseNumber(x)
		if err != nil {
			return err
		}
		row[f.cols[1]] = n
		return nil
	case string:
		if f.kind == fLimitValue && !hasCurrency {
			row[f.cols[2]] = x
			return nil
		}
	case bool:
		if f.kind == fLimitValue && !hasCurrency {
			row[f.cols[3]] = x
			return nil
		}
	}
	return fmt.Errorf("%w: value %v", ErrUnrepresentable, v)
}
