package keel

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
)

// schemaKeyword is a JSON Schema 2020-12 keyword stored in a typed column of json_schema_node.
type schemaKeyword struct {
	col   string
	typ   byte // s string, i non-negative integer, n number, b boolean
	width int  // the column's character limit; 0 is unbounded
}

var schemaKeywords = map[string]schemaKeyword{
	"$schema": {"dialect", 's', 2000}, "$id": {"schema_uri", 's', 2000}, "$anchor": {"anchor", 's', 200},
	"$dynamicAnchor": {"dynamic_anchor", 's', 200}, "$ref": {"ref", 's', 2000}, "$dynamicRef": {"dynamic_ref", 's', 2000},
	"title": {"title", 's', 500}, "description": {"description", 's', 0}, "$comment": {"comment", 's', 0},
	"format": {"format", 's', 120}, "pattern": {"pattern", 's', 2000}, "contentMediaType": {"content_media_type", 's', 120},
	"contentEncoding": {"content_encoding", 's', 60},
	"minLength":       {"min_length", 'i', 0}, "maxLength": {"max_length", 'i', 0}, "minItems": {"min_items", 'i', 0},
	"maxItems": {"max_items", 'i', 0}, "minContains": {"min_contains", 'i', 0}, "maxContains": {"max_contains", 'i', 0},
	"minProperties": {"min_properties", 'i', 0}, "maxProperties": {"max_properties", 'i', 0},
	"minimum": {"minimum", 'n', 0}, "maximum": {"maximum", 'n', 0}, "exclusiveMinimum": {"exclusive_minimum", 'n', 0},
	"exclusiveMaximum": {"exclusive_maximum", 'n', 0}, "multipleOf": {"multiple_of", 'n', 0},
	"uniqueItems": {"unique_items", 'b', 0}, "readOnly": {"read_only", 'b', 0}, "writeOnly": {"write_only", 'b', 0},
	"deprecated": {"deprecated", 'b', 0},
}

// Applicators by how they hold subschemas.
var (
	keyedApplicators   = map[string]bool{"properties": true, "patternProperties": true, "dependentSchemas": true, "$defs": true}
	indexedApplicators = map[string]bool{"prefixItems": true, "allOf": true, "anyOf": true, "oneOf": true}
	singleApplicators  = map[string]bool{"additionalProperties": true, "propertyNames": true, "unevaluatedProperties": true,
		"items": true, "contains": true, "unevaluatedItems": true, "not": true, "if": true, "then": true, "else": true, "contentSchema": true}
)

// schemaNode is one subschema as json_schema_node and its child tables hold it.
type schemaNode struct {
	boolean     *bool
	columns     map[string]any
	constValue  *jsonTree
	defaultVal  *jsonTree
	types       []string
	required    []string
	dependent   [][2]string
	enum        []any
	examples    []any
	vocabulary  map[string]bool
	annotations map[string]any
	children    []schemaChild
}

type schemaChild struct {
	applicator string // code
	key        *string
	index      *int64
	node       *schemaNode
}

func schemaLike(v any) bool {
	switch v.(type) {
	case bool, map[string]any:
		return true
	}
	return false
}

// decodeSchema splits a schema into typed keywords, child tables, and subschemas. A keyword whose value the typed form
// cannot hold exactly, or that the 2020-12 vocabularies do not define, is kept whole as an annotation.
func decodeSchema(v any) (*schemaNode, error) {
	switch x := v.(type) {
	case bool:
		return &schemaNode{boolean: &x}, nil
	case map[string]any:
		n := &schemaNode{columns: map[string]any{}, annotations: map[string]any{}}
		for k, val := range x {
			ok, err := n.keyword(k, val)
			if err != nil {
				return nil, err
			}
			if !ok {
				n.annotations[k] = val
			}
		}
		return n, nil
	}
	return nil, fmt.Errorf("%w: a JSON Schema is an object or a boolean", ErrUnrepresentable)
}

// keyword stores one keyword in typed form; false keeps it as an annotation.
func (n *schemaNode) keyword(k string, val any) (bool, error) {
	if kw, ok := schemaKeywords[k]; ok {
		switch kw.typ {
		case 's':
			s, ok := val.(string)
			if !ok || (kw.width > 0 && len([]rune(s)) > kw.width) {
				return false, nil
			}
			n.columns[kw.col] = s
		case 'i':
			num, ok := val.(json.Number)
			if !ok {
				return false, nil
			}
			i, err := strconv.ParseInt(string(num), 10, 64)
			if err != nil || i < 0 || i > math.MaxInt32 {
				return false, nil
			}
			n.columns[kw.col] = i
		case 'n':
			num, err := parseNumber(val)
			if err != nil {
				return false, nil
			}
			n.columns[kw.col] = num
		case 'b':
			b, ok := val.(bool)
			if !ok {
				return false, nil
			}
			n.columns[kw.col] = b
		}
		return true, nil
	}
	switch {
	case k == "const":
		n.constValue = &jsonTree{Value: val}
	case k == "default":
		n.defaultVal = &jsonTree{Value: val}
	case k == "type":
		types, ok := uniqueStrings(val, true)
		if !ok {
			return false, nil
		}
		for _, t := range types {
			c, err := jsonTypeCodes.code(t)
			if err != nil {
				return false, nil
			}
			n.types = append(n.types, c)
		}
	case k == "required":
		names, ok := uniqueStrings(val, false)
		if !ok {
			return false, nil
		}
		n.required = names
	case k == "dependentRequired":
		m, ok := val.(map[string]any)
		if !ok || len(m) == 0 {
			return false, nil
		}
		var pairs [][2]string
		for prop, list := range m {
			names, ok := uniqueStrings(list, false)
			if !ok {
				return false, nil
			}
			for _, r := range names {
				pairs = append(pairs, [2]string{prop, r})
			}
		}
		n.dependent = pairs
	case k == "enum" || k == "examples":
		list, ok := val.([]any)
		if !ok || len(list) == 0 {
			return false, nil
		}
		if k == "enum" {
			n.enum = list
		} else {
			n.examples = list
		}
	case k == "$vocabulary":
		m, ok := val.(map[string]any)
		if !ok || len(m) == 0 {
			return false, nil
		}
		vocab := map[string]bool{}
		for uri, req := range m {
			b, ok := req.(bool)
			if !ok {
				return false, nil
			}
			vocab[uri] = b
		}
		n.vocabulary = vocab
	case keyedApplicators[k]:
		m, ok := val.(map[string]any)
		if !ok || len(m) == 0 {
			return false, nil
		}
		for _, sub := range m {
			if !schemaLike(sub) {
				return false, nil
			}
		}
		for key, sub := range m {
			child, err := decodeSchema(sub)
			if err != nil {
				return false, err
			}
			key := key
			n.children = append(n.children, schemaChild{applicator: applicatorCodes.codes[k], key: &key, node: child})
		}
	case indexedApplicators[k]:
		list, ok := val.([]any)
		if !ok || len(list) == 0 {
			return false, nil
		}
		for _, sub := range list {
			if !schemaLike(sub) {
				return false, nil
			}
		}
		for i, sub := range list {
			child, err := decodeSchema(sub)
			if err != nil {
				return false, err
			}
			index := int64(i)
			n.children = append(n.children, schemaChild{applicator: applicatorCodes.codes[k], index: &index, node: child})
		}
	case singleApplicators[k]:
		if !schemaLike(val) {
			return false, nil
		}
		child, err := decodeSchema(val)
		if err != nil {
			return false, err
		}
		n.children = append(n.children, schemaChild{applicator: applicatorCodes.codes[k], node: child})
	default:
		return false, nil
	}
	return true, nil
}

// uniqueStrings reads a non-empty array of distinct strings, or a single string when single is set.
func uniqueStrings(v any, single bool) ([]string, bool) {
	if s, ok := v.(string); ok && single {
		return []string{s}, true
	}
	list, ok := v.([]any)
	if !ok || len(list) == 0 {
		return nil, false
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(list))
	for _, e := range list {
		s, ok := e.(string)
		if !ok || seen[s] {
			return nil, false
		}
		seen[s] = true
		out = append(out, s)
	}
	return out, true
}

// encode rebuilds the canonical schema: one type as a string, sets in sorted order.
func (n *schemaNode) encode() any {
	if n.boolean != nil {
		return *n.boolean
	}
	out := map[string]any{}
	for k, kw := range schemaKeywords {
		if v, ok := n.columns[kw.col]; ok {
			if i, isInt := v.(int64); isInt {
				v = json.Number(strconv.FormatInt(i, 10))
			}
			out[k] = v
		}
	}
	if n.constValue != nil {
		out["const"] = n.constValue.Value
	}
	if n.defaultVal != nil {
		out["default"] = n.defaultVal.Value
	}
	if len(n.types) > 0 {
		words := make([]string, len(n.types))
		for i, c := range n.types {
			words[i] = jsonTypeCodes.words[c]
		}
		sort.Strings(words)
		if len(words) == 1 {
			out["type"] = words[0]
		} else {
			out["type"] = toAny(words)
		}
	}
	if len(n.required) > 0 {
		names := append([]string(nil), n.required...)
		sort.Strings(names)
		out["required"] = toAny(names)
	}
	if len(n.dependent) > 0 {
		m := map[string][]string{}
		for _, p := range n.dependent {
			m[p[0]] = append(m[p[0]], p[1])
		}
		dep := map[string]any{}
		for k, names := range m {
			sort.Strings(names)
			dep[k] = toAny(names)
		}
		out["dependentRequired"] = dep
	}
	if len(n.enum) > 0 {
		out["enum"] = n.enum
	}
	if len(n.examples) > 0 {
		out["examples"] = n.examples
	}
	if len(n.vocabulary) > 0 {
		vocab := map[string]any{}
		for uri, req := range n.vocabulary {
			vocab[uri] = req
		}
		out["$vocabulary"] = vocab
	}
	for k, v := range n.annotations {
		out[k] = v
	}
	indexed := map[string][]schemaChild{}
	for _, c := range n.children {
		word := applicatorCodes.words[c.applicator]
		switch {
		case c.key != nil:
			m, _ := out[word].(map[string]any)
			if m == nil {
				m = map[string]any{}
				out[word] = m
			}
			m[*c.key] = c.node.encode()
		case c.index != nil:
			indexed[word] = append(indexed[word], c)
		default:
			out[word] = c.node.encode()
		}
	}
	for word, list := range indexed {
		sort.Slice(list, func(i, j int) bool { return *list[i].index < *list[j].index })
		items := make([]any, len(list))
		for i, c := range list {
			items[i] = c.node.encode()
		}
		out[word] = items
	}
	return out
}

func toAny(s []string) []any {
	out := make([]any, len(s))
	for i, x := range s {
		out[i] = x
	}
	return out
}

// jsonValueRow is one json_value node.
type jsonValueRow struct {
	id, parent  int64
	member      *string
	index       *int64
	valueType   string
	scalarValue *string
}

// jsonType classifies a decoded JSON value by its json_value code and scalar text.
func jsonType(v any) (string, *string, error) {
	text := func(s string) *string { return &s }
	switch x := v.(type) {
	case nil:
		return "Z", nil, nil
	case map[string]any:
		return "O", nil, nil
	case []any:
		return "A", nil, nil
	case string:
		return "S", text(x), nil
	case json.Number:
		return "N", text(string(x)), nil
	case bool:
		return "B", text(strconv.FormatBool(x)), nil
	}
	return "", nil, fmt.Errorf("%w: %T is not a JSON value", ErrUnrepresentable, v)
}

// buildJSON assembles a value from its node rows.
func buildJSON(id int64, nodes map[int64]*jsonValueRow, children map[int64][]*jsonValueRow) (any, error) {
	n, ok := nodes[id]
	if !ok {
		return nil, fmt.Errorf("%w: json_value %d", ErrDocumentIncomplete, id)
	}
	scalar := func() (string, error) {
		if n.scalarValue == nil {
			return "", fmt.Errorf("%w: json_value %d has no scalar", ErrDocumentIncomplete, id)
		}
		return *n.scalarValue, nil
	}
	switch n.valueType {
	case "Z":
		return nil, nil
	case "S":
		return scalar()
	case "N":
		s, err := scalar()
		return json.Number(s), err
	case "B":
		s, err := scalar()
		if err != nil {
			return nil, err
		}
		return strconv.ParseBool(s)
	case "O":
		out := map[string]any{}
		for _, c := range children[id] {
			if c.member == nil {
				return nil, fmt.Errorf("%w: json_value %d member has no name", ErrDocumentIncomplete, c.id)
			}
			v, err := buildJSON(c.id, nodes, children)
			if err != nil {
				return nil, err
			}
			out[*c.member] = v
		}
		return out, nil
	case "A":
		list := children[id]
		for _, c := range list {
			if c.index == nil {
				return nil, fmt.Errorf("%w: json_value %d item has no index", ErrDocumentIncomplete, c.id)
			}
		}
		sort.Slice(list, func(i, j int) bool { return *list[i].index < *list[j].index })
		out := make([]any, len(list))
		for i, c := range list {
			if *c.index != int64(i) {
				return nil, fmt.Errorf("%w: json_value %d items are not contiguous", ErrDocumentIncomplete, id)
			}
			v, err := buildJSON(c.id, nodes, children)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	}
	return nil, fmt.Errorf("%w: json_value %d type %q", ErrDocumentIncomplete, id, n.valueType)
}
