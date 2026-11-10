package keel

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/nauticana/charter/sdk/model"
)

// composer rebuilds one document from its logical rows.
type composer struct {
	ns       string
	rows     lrows
	kindOf   map[objKey]model.Kind
	exponent map[string]int64
	agent    *objKey
}

// compose writes the canonical document: sets in key order, sequences in ordinal order, same-namespace references
// as strings, empty optional lists absent, and required lists present even when empty.
func (c *composer) compose(spec *kindSpec, id string, enterprise *objKey) (map[string]any, error) {
	doc := map[string]any{"namespace": c.ns, "id": id, "kind": string(spec.kind)}
	if enterprise != nil {
		doc["enterpriseId"] = formatIDRef(*enterprise, c.ns)
	}
	for _, t := range []*tableSpec{spec.envelope, spec.identity, spec.root} {
		if t == nil {
			continue
		}
		rows := matching(c.rows[t.name], t.owner, topOwner(t))
		if len(rows) != 1 {
			return nil, fmt.Errorf("%w: %s has %d %s rows", ErrDocumentIncomplete, id, len(rows), t.name)
		}
		if err := c.table(t, rows[0], doc); err != nil {
			return nil, err
		}
	}
	for _, p := range spec.objects {
		if _, ok := lookupPath(doc, p); !ok {
			setPath(doc, p, map[string]any{})
		}
	}
	return doc, nil
}

func lookupPath(m map[string]any, path string) (any, bool) {
	var v any = m
	for _, p := range splitPath(path) {
		obj, ok := v.(map[string]any)
		if !ok {
			return nil, false
		}
		if v, ok = obj[p]; !ok {
			return nil, false
		}
	}
	return v, true
}

func matching(rows []lrow, cols []string, values []any) []lrow {
	want := rowKey(values...)
	var out []lrow
	for _, r := range rows {
		got := make([]any, len(cols))
		for i, c := range cols {
			got[i] = r[c]
		}
		if rowKey(got...) == want {
			out = append(out, r)
		}
	}
	return out
}

func (c *composer) table(t *tableSpec, row lrow, item map[string]any) error {
	for _, f := range t.fields {
		if err := c.field(f, row, item); err != nil {
			return fmt.Errorf("%s %s: %w", t.name, f.path, err)
		}
	}
	pk := t.pk(row)
	for i := range t.lists {
		if err := c.list(&t.lists[i], pk, item); err != nil {
			return err
		}
	}
	return nil
}

func (c *composer) list(l *listSpec, parentPK []any, item map[string]any) error {
	rows := matching(c.rows[l.table.name], l.table.owner, childOwner(l, parentPK))
	t := l.table
	switch l.shape {
	case lOne:
		if len(rows) == 0 {
			return nil
		}
		m := map[string]any{}
		if err := c.table(t, rows[0], m); err != nil {
			return err
		}
		setPath(item, l.path, m)
		return nil
	case lMap:
		if len(rows) == 0 {
			return nil
		}
		m := map[string]any{}
		for _, r := range rows {
			v, _, err := c.value(t.fields[0], r)
			if err != nil {
				return err
			}
			name, _ := r[l.mapKey].(string)
			m[name] = v
		}
		setPath(item, l.path, m)
		return nil
	}
	if t.order != "" {
		sort.SliceStable(rows, func(i, j int) bool { return asInt(rows[i][t.order]) < asInt(rows[j][t.order]) })
	}
	items := make([]any, 0, len(rows))
	for _, r := range rows {
		switch {
		case l.shape == lSplit:
			key, ok := r[colSplitRef].(objKey)
			if !ok {
				return fmt.Errorf("%w: %s member", ErrDocumentIncomplete, l.path)
			}
			items = append(items, formatIDRef(key, c.ns))
		case l.scalar:
			v, _, err := c.value(t.fields[0], r)
			if err != nil {
				return err
			}
			items = append(items, v)
		case t.prose != "" && r[t.form] == nil:
			items = append(items, r[t.prose])
		default:
			m := map[string]any{}
			if err := c.table(t, r, m); err != nil {
				return err
			}
			items = append(items, m)
		}
	}
	if t.order == "" {
		sort.SliceStable(items, func(i, j int) bool { return canonicalJSON(items[i]) < canonicalJSON(items[j]) })
	}
	if len(items) > 0 || l.always {
		setPath(item, l.path, items)
	}
	return nil
}

func asInt(v any) int64 {
	i, _ := v.(int64)
	return i
}

func (c *composer) field(f field, row lrow, item map[string]any) error {
	if f.kind == fPin {
		key, ok := row[f.col].(objKey)
		if !ok {
			return nil
		}
		label, ok := row[f.verCol].(versionLabel)
		if !ok {
			return fmt.Errorf("%w: %s has no version label", ErrDocumentIncomplete, f.path)
		}
		setPath(item, f.path, formatIDRef(key, c.ns))
		setPath(item, f.label, label.Label)
		return nil
	}
	v, ok, err := c.value(f, row)
	if err != nil || !ok {
		return err
	}
	setPath(item, f.path, v)
	return nil
}

// value is the JSON value a field's columns hold; ok is false when the property is absent.
func (c *composer) value(f field, r lrow) (any, bool, error) {
	switch f.kind {
	case fText, fDate:
		v, ok := r[f.col].(string)
		return v, ok, nil
	case fInt:
		v, ok := r[f.col].(int64)
		return v, ok, nil
	case fNumber:
		v, ok := r[f.col].(json.Number)
		return v, ok, nil
	case fBool:
		v, ok := r[f.col].(bool)
		return v, ok, nil
	case fFlag:
		v, _ := r[f.col].(bool)
		return true, v, nil
	case fTime:
		v, ok := r[f.col].(time.Time)
		if !ok {
			return nil, false, nil
		}
		return formatTime(v), true, nil
	case fCode:
		code, ok := r[f.col].(string)
		if !ok {
			return nil, false, nil
		}
		word, err := f.codes.word(code)
		return word, err == nil, err
	case fCodeOrText:
		if code, ok := r[f.cols[0]].(string); ok {
			word, err := f.codes.word(code)
			return word, err == nil, err
		}
		v, ok := r[f.cols[1]].(string)
		return v, ok, nil
	case fRef:
		key, ok := r[f.col].(objKey)
		if !ok {
			return nil, false, nil
		}
		return formatIDRef(key, c.ns), true, nil
	case fObject:
		key, ok := r[f.col].(objKey)
		if !ok {
			return nil, false, nil
		}
		kind, known := c.kindOf[key]
		if !known && key.Ext == "" {
			return nil, false, fmt.Errorf("%w: kind of %s", ErrDocumentIncomplete, key)
		}
		return formatObjectRef(key, kind, c.ns), true, nil
	case fArc:
		for _, a := range f.arcs {
			if key, ok := r[a.col].(objKey); ok {
				return formatObjectRef(key, a.kind, c.ns), true, nil
			}
		}
		return nil, false, nil
	case fRefArc:
		key, ok := r["#"+f.path].(objKey)
		if !ok {
			return nil, false, nil
		}
		return formatIDRef(key, c.ns), true, nil
	case fJSON:
		v, ok := r[f.col].(jsonTree)
		return v.Value, ok, nil
	case fSchema:
		v, ok := r[f.col].(schemaTree)
		return v.Value, ok, nil
	case fConst:
		return f.value, true, nil
	case fApproved:
		v, ok := r[colApproved].(string)
		return v, ok, nil
	case fMeasured:
		v, ok := r[colMeasured].(json.Number)
		return v, ok, nil
	case fMoney, fLimitValue:
		if i, ok := r[f.cols[0]].(int64); ok {
			return json.Number(strconv.FormatInt(i, 10)), true, nil
		}
		if n, ok := r[f.cols[1]].(json.Number); ok {
			return n, true, nil
		}
		if f.kind == fLimitValue {
			if s, ok := r[f.cols[2]].(string); ok {
				return s, true, nil
			}
			if b, ok := r[f.cols[3]].(bool); ok {
				return b, true, nil
			}
		}
		return nil, false, nil
	case fExponent:
		code, ok := r["currency_code"].(string)
		if !ok {
			return nil, false, nil
		}
		exp, known := c.exponent[code]
		if !known {
			return nil, false, fmt.Errorf("%w: currency %s is not in keel's currency table", ErrCurrencyExponent, code)
		}
		return exp, true, nil
	case fActor:
		if key, ok := r[f.col].(objKey); ok {
			return formatObjectRef(key, model.KindHumanIdentity, c.ns), true, nil
		}
		if c.agent == nil {
			return nil, false, fmt.Errorf("%w: the action has neither a human actor nor a runtime identity", ErrDocumentIncomplete)
		}
		return formatObjectRef(*c.agent, model.KindAgentIdentity, c.ns), true, nil
	case fRuntimeAgent:
		if c.agent == nil {
			return nil, false, fmt.Errorf("%w: the runtime's definition has no identity", ErrDocumentIncomplete)
		}
		return formatIDRef(*c.agent, c.ns), true, nil
	}
	return nil, false, fmt.Errorf("%w: field kind %d", ErrDocumentIncomplete, f.kind)
}
