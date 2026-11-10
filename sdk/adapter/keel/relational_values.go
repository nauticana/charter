package keel

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nauticana/charter/sdk/model"
)

var (
	ErrUnmappedProperty   = errors.New("document property has no column")
	ErrUnrepresentable    = errors.New("document value cannot be stored without loss")
	ErrUnresolvedRef      = errors.New("referenced object is not stored")
	ErrFixedChanged       = errors.New("a property fixed when the document was first stored cannot change")
	ErrRecordWritten      = errors.New("a stored record cannot change; a correction supersedes it")
	ErrUnknownVersion     = errors.New("no stored definition version has this label")
	ErrCurrencyExponent   = errors.New("currency exponent disagrees with the currency")
	ErrRuntimeActor       = errors.New("an agent acts only as its runtime's identity")
	ErrDocumentIncomplete = errors.New("stored document rows are incomplete")
	ErrModuleNotInstalled = errors.New("the schema module is not installed")
)

// objKey names a referenced object: a document by namespace and identifier, or an object outside the document set by
// its external kind as well.
type objKey struct{ NS, ID, Ext string }

func (k objKey) String() string {
	if k.Ext != "" {
		return k.Ext + " " + k.NS + ":" + k.ID
	}
	return k.NS + ":" + k.ID
}

// selfID and selfVersion stand for the owning document's object id and version number in a logical row, so the rows
// of a document compare equal however it was stored.
type (
	selfID      struct{}
	selfVersion struct{}
)

// versionLabel is a pinned definition version: the definition and the label it declares.
type versionLabel struct {
	Object objKey
	Label  string
}

// jsonTree and schemaTree are values stored as json_value and json_schema_node trees, held in canonical form.
type (
	jsonTree   struct{ Value any }
	schemaTree struct{ Value any }
)

// lrow is a logical row: column values before object ids, version numbers, and tree roots are resolved.
type lrow map[string]any

// lrows holds a document's logical rows by table; a name starting with # is a list whose table depends on the kind
// each member resolves to.
type lrows map[string][]lrow

const (
	colApproved = "#approved_action"
	colMeasured = "#measured_value"
	colSplitRef = "#ref"
)

// rowKey renders values comparably: ids, numbers, times, and trees in canonical form.
func rowKey(values ...any) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = canonicalValue(v)
	}
	return strings.Join(parts, "\x1f")
}

func canonicalValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "~"
	case selfID:
		return "#self"
	case selfVersion:
		return "#version"
	case objKey:
		return "@" + x.String()
	case versionLabel:
		return "@" + x.Object.String() + "@" + x.Label
	case json.Number:
		if r, ok := new(big.Rat).SetString(string(x)); ok {
			return "n" + r.RatString()
		}
		return "n" + string(x)
	case time.Time:
		return "t" + x.UTC().Format(time.RFC3339Nano)
	case jsonTree:
		return "j" + canonicalJSON(x.Value)
	case schemaTree:
		return "s" + canonicalJSON(x.Value)
	case int64:
		return "i" + strconv.FormatInt(x, 10)
	case bool:
		return "b" + strconv.FormatBool(x)
	case string:
		return "'" + x
	}
	return fmt.Sprintf("?%v", v)
}

// canonicalJSON encodes a value with sorted members and numbers in normalized form.
func canonicalJSON(v any) string {
	var b strings.Builder
	writeCanonical(&b, v)
	return b.String()
}

func writeCanonical(b *strings.Builder, v any) {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.Quote(k))
			b.WriteByte(':')
			writeCanonical(b, x[k])
		}
		b.WriteByte('}')
	case []any:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			writeCanonical(b, e)
		}
		b.WriteByte(']')
	case json.Number:
		b.WriteString(canonicalValue(x)[1:])
	default:
		raw, _ := json.Marshal(x)
		b.Write(raw)
	}
}

// sameRows reports whether two documents' logical rows are equal as sets per table.
func sameRows(a, b lrows) bool {
	tables := map[string]bool{}
	for t := range a {
		tables[t] = true
	}
	for t := range b {
		tables[t] = true
	}
	for t := range tables {
		if !slices.Equal(rowSet(a[t]), rowSet(b[t])) {
			return false
		}
	}
	return true
}

func rowSet(rows []lrow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		keys := make([]string, 0, len(r))
		for k, v := range r {
			if v != nil {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for j, k := range keys {
			parts[j] = k + "=" + canonicalValue(r[k])
		}
		out[i] = strings.Join(parts, "\x1e")
	}
	sort.Strings(out)
	return out
}

// splitPath splits a dot path; the empty path names the value itself.
func splitPath(path string) []string {
	if path == "" {
		return nil
	}
	return strings.Split(path, ".")
}

// takePath removes and returns the value at path, dropping parent objects it leaves empty.
func takePath(m map[string]any, path string) (any, bool) {
	parts := splitPath(path)
	if len(parts) == 0 {
		return nil, false
	}
	parent, ok := m, true
	trail := []map[string]any{m}
	for _, p := range parts[:len(parts)-1] {
		parent, ok = parent[p].(map[string]any)
		if !ok {
			return nil, false
		}
		trail = append(trail, parent)
	}
	last := parts[len(parts)-1]
	v, ok := parent[last]
	if !ok {
		return nil, false
	}
	delete(parent, last)
	for i := len(trail) - 1; i > 0; i-- {
		if len(trail[i]) > 0 {
			break
		}
		delete(trail[i-1], parts[i-1])
	}
	return v, true
}

// setPath stores v at path, creating parent objects.
func setPath(m map[string]any, path string, v any) {
	parts := splitPath(path)
	for _, p := range parts[:len(parts)-1] {
		next, ok := m[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			m[p] = next
		}
		m = next
	}
	m[parts[len(parts)-1]] = v
}

// leftover lists the paths of m no column consumed.
func leftover(m map[string]any, prefix string) []string {
	var out []string
	for k, v := range m {
		path := prefix + k
		if sub, ok := v.(map[string]any); ok && len(sub) > 0 {
			out = append(out, leftover(sub, path+".")...)
			continue
		}
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

// cloneJSON copies a decoded JSON value so decomposition can consume it.
func cloneJSON(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			out[k] = cloneJSON(e)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = cloneJSON(e)
		}
		return out
	}
	return v
}

// parseIDRef reads an idRef: a same-namespace identifier string or a namespaced object.
func parseIDRef(v any, ns string) (objKey, error) {
	switch x := v.(type) {
	case string:
		if x != "" {
			return objKey{NS: ns, ID: x}, nil
		}
	case map[string]any:
		id, idOK := x["id"].(string)
		namespace, nsOK := x["namespace"].(string)
		if idOK && nsOK && id != "" && namespace != "" && len(x) == 2 {
			return objKey{NS: namespace, ID: id}, nil
		}
	}
	return objKey{}, fmt.Errorf("%w: malformed idRef %v", ErrUnrepresentable, v)
}

// parseObjectRef reads an objectRef; an external target keeps its kind as the external kind.
func parseObjectRef(v any, ns string) (objKey, model.Kind, error) {
	x, ok := v.(map[string]any)
	if !ok {
		return objKey{}, "", fmt.Errorf("%w: malformed objectRef %v", ErrUnrepresentable, v)
	}
	kind, kindOK := x["kind"].(string)
	id, idOK := x["id"].(string)
	if !kindOK || !idOK || kind == "" || id == "" {
		return objKey{}, "", fmt.Errorf("%w: malformed objectRef %v", ErrUnrepresentable, v)
	}
	key := objKey{NS: ns, ID: id}
	for k, e := range x {
		switch k {
		case "kind", "id":
		case "namespace":
			s, ok := e.(string)
			if !ok || s == "" {
				return objKey{}, "", fmt.Errorf("%w: malformed objectRef namespace %v", ErrUnrepresentable, e)
			}
			key.NS = s
		case "external":
			b, ok := e.(bool)
			if !ok {
				return objKey{}, "", fmt.Errorf("%w: malformed objectRef external %v", ErrUnrepresentable, e)
			}
			if b {
				key.Ext = kind
			}
		default:
			return objKey{}, "", fmt.Errorf("%w: objectRef member %s", ErrUnmappedProperty, k)
		}
	}
	return key, model.Kind(kind), nil
}

// formatIDRef writes the string form for a target in the document's own namespace.
func formatIDRef(k objKey, ns string) any {
	if k.NS == ns {
		return k.ID
	}
	return map[string]any{"namespace": k.NS, "id": k.ID}
}

func formatObjectRef(k objKey, kind model.Kind, ns string) map[string]any {
	out := map[string]any{"kind": string(kind), "id": k.ID}
	if k.NS != ns {
		out["namespace"] = k.NS
	}
	if k.Ext != "" {
		out["kind"] = k.Ext
		out["external"] = true
	}
	return out
}

// parseTime reads a date-time at the microsecond precision a TIMESTAMP keeps.
func parseTime(v any) (time.Time, error) {
	s, ok := v.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("%w: date-time %v is not a string", ErrUnrepresentable, v)
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: date-time %q: %v", ErrUnrepresentable, s, err)
	}
	t = t.UTC()
	if !t.Equal(t.Truncate(time.Microsecond)) {
		return time.Time{}, fmt.Errorf("%w: date-time %q is finer than a microsecond", ErrUnrepresentable, s)
	}
	return t, nil
}

func formatTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

const dateLayout = "2006-01-02"

func parseDate(v any) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%w: date %v is not a string", ErrUnrepresentable, v)
	}
	if _, err := time.Parse(dateLayout, s); err != nil {
		return "", fmt.Errorf("%w: date %q: %v", ErrUnrepresentable, s, err)
	}
	return s, nil
}

// parseInt reads a JSON integer.
func parseInt(v any) (int64, error) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, fmt.Errorf("%w: %v is not a number", ErrUnrepresentable, v)
	}
	i, err := strconv.ParseInt(string(n), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %s is not a 64-bit integer", ErrUnrepresentable, n)
	}
	return i, nil
}

func parseNumber(v any) (json.Number, error) {
	n, ok := v.(json.Number)
	if !ok {
		return "", fmt.Errorf("%w: %v is not a number", ErrUnrepresentable, v)
	}
	if _, ok := new(big.Rat).SetString(string(n)); !ok {
		return "", fmt.Errorf("%w: %s is not a finite number", ErrUnrepresentable, n)
	}
	return n, nil
}
