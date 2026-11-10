package keel

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"math/big"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/evidence/evidencetest"
	"github.com/nauticana/charter/sdk/model"
	charterschema "github.com/nauticana/charter/sdk/schema"
)

// mappedElsewhere are tables the kind mappings do not own: object identity, JSON trees, and account links.
var mappedElsewhere = []string{"enterprise_object", "json_value", "json_schema_node", "json_schema_type", "json_schema_required",
	"json_schema_dependent_required", "json_schema_enum", "json_schema_example", "json_schema_vocabulary", "json_schema_annotation",
	"human_identity_partner_user"}

// TestKindMappingsCoverTheSchema checks that every table of the schema modules is written by exactly one mechanism and
// every column of a mapped table holds some document property, key, or order.
func TestKindMappingsCoverTheSchema(t *testing.T) {
	m, err := relational()
	if err != nil {
		t.Fatal(err)
	}
	if len(kindSpecs) != 46 {
		t.Errorf("%d kinds are mapped, want 46", len(kindSpecs))
	}
	covered := map[string]map[string]bool{}
	mark := func(table string, cols ...string) {
		if covered[table] == nil {
			covered[table] = map[string]bool{"partner_id": true}
		}
		for _, c := range cols {
			covered[table][c] = true
		}
	}
	for _, spec := range kindSpecs {
		spec.tables(func(ts *tableSpec, l *listSpec) {
			if l != nil {
				for _, target := range l.targets {
					mark(target.table, append(slices.Clone(ts.owner), target.col)...)
				}
				return
			}
			mark(ts.name, ts.owner...)
			mark(ts.name, ts.key...)
			mark(ts.name, ts.order, ts.prose, ts.form)
			for _, f := range ts.fields {
				mark(ts.name, f.columns()...)
			}
		})
	}
	for _, name := range m.cat.order {
		if slices.Contains(mappedElsewhere, name) {
			if covered[name] != nil {
				t.Errorf("%s is mapped by a kind and handled elsewhere", name)
			}
			continue
		}
		if covered[name] == nil {
			t.Errorf("no kind writes %s", name)
			continue
		}
		for _, c := range m.cat.tables[name].written() {
			if !covered[name][c.Name] {
				t.Errorf("%s.%s holds no document property", name, c.Name)
			}
		}
	}
}

// TestCodeListsMatchTheSeed keeps the Go code lists, the modules' seeded constant values, and their constant lookups
// equal; each list is seeded once, and a module looks up only lists it or a module it depends on seeds.
func TestCodeListsMatchTheSeed(t *testing.T) {
	cat, err := relationalCatalog()
	if err != nil {
		t.Fatal(err)
	}
	values, lookups := map[string][]string{}, map[string]string{}
	owner := map[string]string{}
	lookupModule := map[string]string{}
	for _, m := range cat.modules {
		raw, err := fs.ReadFile(charterschema.Modules, "seed/"+m.Path+".yml")
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		var seed struct {
			Seeds []struct {
				Table string  `yaml:"table"`
				Rows  [][]any `yaml:"rows"`
			} `yaml:"seeds"`
		}
		if err := yaml.Unmarshal(raw, &seed); err != nil {
			t.Fatal(err)
		}
		for _, s := range seed.Seeds {
			for _, row := range s.Rows {
				switch s.Table {
				case "constant_header":
					if prior, seen := owner[row[0].(string)]; seen {
						t.Errorf("%s is seeded by %s and %s", row[0], prior, m.Name)
					}
					owner[row[0].(string)] = m.Name
				case "constant_value":
					values[row[0].(string)] = append(values[row[0].(string)], row[1].(string))
				case "constant_lookup":
					column := row[1].(string) + "." + row[2].(string)
					lookups[column], lookupModule[column] = row[0].(string), m.Name
					if cat.tables[row[1].(string)].module != m.Name {
						t.Errorf("%s seeds the lookup of %s, a column of %s", m.Name, column, cat.tables[row[1].(string)].module)
					}
				}
			}
		}
	}
	reaches := func(from, to string) bool {
		seen, stack := map[string]bool{}, []string{from}
		for len(stack) > 0 {
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if n == to {
				return true
			}
			for _, m := range cat.modules {
				if m.Name == n && !seen[n] {
					seen[n] = true
					stack = append(stack, m.DependsOn...)
				}
			}
		}
		return false
	}
	for column, id := range lookups {
		if !reaches(lookupModule[column], owner[id]) {
			t.Errorf("%s looks up %s, which %s seeds outside its dependencies", lookupModule[column], id, owner[id])
		}
	}
	for _, c := range allCodeLists {
		got, want := slices.Clone(c.order), slices.Clone(values[c.id])
		sort.Strings(got)
		sort.Strings(want)
		if !slices.Equal(got, want) {
			t.Errorf("%s codes %v, seed %v", c.id, got, want)
		}
		delete(values, c.id)
	}
	for id := range values {
		t.Errorf("seed list %s has no Go code list", id)
	}
	used := map[string]string{"enterprise_object.kind": kindCodes.id, "json_value.value_type": jsonValueTypeCodes.id,
		"json_schema_type.json_type": jsonTypeCodes.id, "json_schema_node.applicator": applicatorCodes.id}
	for _, spec := range kindSpecs {
		spec.tables(func(ts *tableSpec, _ *listSpec) {
			for _, f := range ts.fields {
				if f.codes != nil {
					used[ts.name+"."+f.columns()[0]] = f.codes.id
				}
			}
		})
	}
	for column, id := range used {
		if lookups[column] != id {
			t.Errorf("%s reads %s, the seed looks up %q", column, id, lookups[column])
		}
	}
	for column := range lookups {
		if _, ok := used[column]; !ok {
			t.Errorf("seed looks up %s, which no mapping reads", column)
		}
	}
}

// storeAll saves documents and appends evidence in reference order; documents that reference one another are saved
// together.
func storeAll(t *testing.T, store *DocumentStore, partnerID int64, docs []*model.Document) {
	t.Helper()
	ctx := context.Background()
	keys := map[objKey]*model.Document{}
	for _, d := range docs {
		keys[objKey{NS: d.Namespace, ID: d.ID}] = d
	}
	refs := map[*model.Document][]objKey{}
	for _, d := range docs {
		rows, err := decompose(d)
		if err != nil {
			t.Fatalf("%s: %v", d.ID, err)
		}
		refs[d] = referencesOf(rows, keys)
	}
	stored := map[objKey]bool{}
	pending := slices.Clone(docs)
	for len(pending) > 0 {
		ready := func(d *model.Document, also map[objKey]bool) bool {
			for _, r := range refs[d] {
				if !stored[r] && !also[r] && r != (objKey{NS: d.Namespace, ID: d.ID}) {
					return false
				}
			}
			return true
		}
		var batch, appendable []*model.Document
		for _, d := range pending {
			if ready(d, nil) {
				if slices.Contains(evidenceKinds, d.Kind) {
					appendable = append(appendable, d)
				} else {
					batch = append(batch, d)
				}
			}
		}
		if len(batch) == 0 && len(appendable) == 0 {
			together := map[objKey]bool{}
			for _, d := range pending {
				if !slices.Contains(evidenceKinds, d.Kind) {
					together[objKey{NS: d.Namespace, ID: d.ID}] = true
				}
			}
			for _, d := range pending {
				if together[objKey{NS: d.Namespace, ID: d.ID}] && ready(d, together) {
					batch = append(batch, d)
				}
			}
			if len(batch) == 0 {
				if d, trial := withoutPendingEvidence(t, pending, refs, stored, keys); d != nil {
					if err := store.Save(ctx, partnerID, trial); err != nil {
						t.Fatal(err)
					}
					stored[objKey{NS: d.Namespace, ID: d.ID}] = true
					continue
				}
				var blocked []string
				for _, d := range pending {
					var missing []string
					for _, r := range refs[d] {
						if !stored[r] {
							missing = append(missing, r.ID)
						}
					}
					blocked = append(blocked, d.ID+" needs "+strings.Join(missing, ","))
				}
				t.Fatalf("documents reference one another through evidence: %v", blocked)
			}
		}
		if len(batch) > 0 {
			if err := store.Save(ctx, partnerID, batch...); err != nil {
				t.Fatal(err)
			}
		}
		for _, d := range appendable {
			if err := store.Evidence(partnerID).Append(ctx, d); err != nil {
				t.Fatalf("%s: %v", d.ID, err)
			}
		}
		done := map[*model.Document]bool{}
		for _, d := range append(batch, appendable...) {
			stored[objKey{NS: d.Namespace, ID: d.ID}] = true
			done[d] = true
		}
		pending = slices.DeleteFunc(pending, func(d *model.Document) bool { return done[d] })
	}
}

// withoutPendingEvidence breaks a cycle through evidence, such as a measure whose baseline cites a measurement of
// itself: it returns a pending document and a copy without the optional properties that name pending evidence, which
// is stored first and completed in place once the evidence is appended.
func withoutPendingEvidence(t *testing.T, pending []*model.Document, refs map[*model.Document][]objKey, stored map[objKey]bool,
	set map[objKey]*model.Document) (*model.Document, *model.Document) {
	t.Helper()
	for _, d := range pending {
		if slices.Contains(evidenceKinds, d.Kind) || stored[objKey{NS: d.Namespace, ID: d.ID}] {
			continue
		}
		onlyEvidence := true
		for _, r := range refs[d] {
			if !stored[r] && !slices.Contains(evidenceKinds, set[r].Kind) {
				onlyEvidence = false
			}
		}
		if !onlyEvidence {
			continue
		}
		value, err := documentValue(d)
		if err != nil {
			t.Fatal(err)
		}
		for property := range value {
			trial := &model.Document{Envelope: d.Envelope, Value: maps.Clone(value)}
			delete(trial.Value.(map[string]any), property)
			rows, err := decompose(trial)
			if err != nil {
				continue
			}
			ready := true
			for _, r := range referencesOf(rows, set) {
				ready = ready && (stored[r] || r == objKey{NS: d.Namespace, ID: d.ID})
			}
			if ready {
				return d, trial
			}
		}
	}
	return nil, nil
}

// referencesOf lists the documents of the set a document's rows name, including the capability an approval names.
func referencesOf(d *docRows, set map[objKey]*model.Document) []objKey {
	var out []objKey
	add := func(k objKey) {
		if _, ok := set[k]; ok && k.Ext == "" {
			out = append(out, k)
		}
	}
	if d.enterprise != nil {
		add(*d.enterprise)
	}
	for _, rows := range d.rows {
		for _, r := range rows {
			for _, v := range r {
				switch x := v.(type) {
				case objKey:
					add(x)
				case versionLabel:
					add(x.Object)
				}
			}
			if s, ok := r[colApproved].(string); ok {
				add(objKey{NS: d.namespace, ID: s})
			}
		}
	}
	for _, a := range d.agents {
		add(a.runtime)
		add(a.agent)
	}
	return out
}

func newRelationalStore(t *testing.T) (*DocumentStore, *memoryDB) {
	t.Helper()
	db := newMemoryDB(t)
	store, err := NewDocumentStore(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	return store, db
}

// TestHarborRoundTrip stores every Harbor document in its tables and reads each back equal in meaning, then checks
// that saving what was read is a replay that writes nothing.
func TestHarborRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, db := newRelationalStore(t)
	all := harbor(t).Documents()
	storeAll(t, store, partner, all)
	source := store.Source(partner)
	for _, want := range all {
		got, err := source.Fetch(ctx, corpus.DocumentKey{Namespace: want.Namespace, ID: want.ID})
		if err != nil {
			t.Errorf("%s: %v", want.ID, err)
			continue
		}
		if a, b := comparable(t, want, want.Namespace), comparable(t, got, want.Namespace); a != b {
			t.Errorf("%s changed in the store:\n want %s\n got  %s", want.ID, a, b)
		}
	}
	for kind := range kindSpecs {
		listed, err := source.List(ctx, kind)
		if err != nil || len(listed) != len(harbor(t).OfKind(kind)) {
			t.Errorf("list %s: %d, %v", kind, len(listed), err)
		}
	}
	var governance []*model.Document
	for _, d := range all {
		if !slices.Contains(evidenceKinds, d.Kind) {
			read, err := source.Fetch(ctx, corpus.DocumentKey{Namespace: d.Namespace, ID: d.ID})
			if err != nil {
				t.Fatal(err)
			}
			governance = append(governance, read)
		}
	}
	clear(db.calls)
	if err := store.Save(ctx, partner, governance...); err != nil {
		t.Fatal(err)
	}
	for name, n := range db.calls {
		if strings.HasSuffix(name, ".insert") || strings.HasSuffix(name, ".update") || strings.Contains(name, ".delete.") {
			t.Errorf("a replay ran %s %d times", name, n)
		}
	}
	definition, err := source.Fetch(ctx, corpus.DocumentKey{Namespace: ns, ID: "AGENTDEF-ORDER-EXCEPTION-COORDINATOR-1"})
	if err != nil {
		t.Fatal(err)
	}
	original, _ := harbor(t).Get(ns, "AGENTDEF-ORDER-EXCEPTION-COORDINATOR-1")
	for _, list := range []string{"triggers", "policies", "escalationConditions"} {
		if a, b := canonicalJSON(original.Value.(map[string]any)[list]), canonicalJSON(definition.Value.(map[string]any)[list]); a != b {
			t.Errorf("%s order changed: %s, %s", list, a, b)
		}
	}
}

// comparable renders a document with the canonical form's choices applied: same-namespace references as strings,
// date-times in UTC, numbers normalized, false external flags and empty lists absent, and arrays as multisets.
func comparable(t *testing.T, d *model.Document, ns string) string {
	t.Helper()
	value, err := documentValue(d)
	if err != nil {
		t.Fatal(err)
	}
	return canonicalJSON(normalizeForCompare(value, ns))
}

func normalizeForCompare(v any, ns string) any {
	switch x := v.(type) {
	case map[string]any:
		if id, ok := x["id"].(string); ok && len(x) == 2 && x["namespace"] == ns {
			return id
		}
		out := map[string]any{}
		for k, e := range x {
			n := normalizeForCompare(e, ns)
			if list, ok := n.([]any); ok && len(list) == 0 {
				continue
			}
			if m, ok := n.(map[string]any); ok && len(m) == 0 {
				continue
			}
			if k == "external" && n == false {
				continue
			}
			out[k] = n
		}
		if _, isRef := out["kind"]; isRef && out["namespace"] == ns {
			delete(out, "namespace")
		}
		if list, ok := out["type"].([]any); ok && len(list) == 1 {
			out["type"] = list[0]
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = normalizeForCompare(e, ns)
		}
		sort.Slice(out, func(i, j int) bool { return canonicalJSON(out[i]) < canonicalJSON(out[j]) })
		return out
	case string:
		if t, err := time.Parse(time.RFC3339Nano, x); err == nil && strings.Contains(x, "T") {
			return t.UTC().Format(time.RFC3339Nano)
		}
		return x
	case json.Number:
		if r, ok := new(big.Rat).SetString(string(x)); ok {
			return json.Number(r.RatString())
		}
	}
	return v
}

func parseDocument(t *testing.T, raw string) *model.Document {
	t.Helper()
	d, err := (corpus.Parser{}).Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// TestEmbeddedSchemasAndExtensionsRoundTrip stores a contract whose input schema uses every kind of keyword, and
// extension values of every JSON type, and reads them back.
func TestEmbeddedSchemasAndExtensionsRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, db := newDocumentStore(t)
	contract := edited(t, store, "CAP-READ-ORDER-EXCEPTION", "contractVersion", "2")
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema", "$id": "urn:example:order",
		"type": []any{"object", "null"}, "required": []any{"orderId", "amount"},
		"properties": map[string]any{
			"orderId": map[string]any{"type": "string", "pattern": "^SO-[0-9]+$", "maxLength": json.Number("40")},
			"amount":  map[string]any{"type": "number", "minimum": json.Number("0"), "multipleOf": json.Number("0.01")},
			"status":  map[string]any{"enum": []any{"open", "held", nil, json.Number("3")}, "default": "open"},
			"lines":   map[string]any{"type": "array", "items": map[string]any{"$ref": "#/$defs/line"}, "minItems": json.Number("1")},
			"note":    true,
		},
		"$defs":                map[string]any{"line": map[string]any{"type": "object", "additionalProperties": false}},
		"oneOf":                []any{map[string]any{"required": []any{"orderId"}}, false},
		"dependentRequired":    map[string]any{"amount": []any{"currency", "orderId"}},
		"x-charter-ref-kinds":  []any{"Task"},
		"x-note":               map[string]any{"owner": "credit", "weight": json.Number("2")},
		"examples":             []any{map[string]any{"orderId": "SO-1"}},
		"const":                map[string]any{"nested": []any{true, json.Number("1.50")}},
		"$comment":             strings.Repeat("c", 600),
		"$vocabulary":          map[string]any{"https://json-schema.org/draft/2020-12/vocab/core": true},
		"title":                strings.Repeat("t", 501),
		"contentMediaType":     "application/json",
		"uniqueItems":          true,
		"additionalProperties": map[string]any{"type": "string"},
	}
	contract.Value.(map[string]any)["inputSchema"] = map[string]any{"jsonSchema": schema}
	contract.Value.(map[string]any)["extensions"] = map[string]any{
		"vendor:sap":            map[string]any{"reservation": "0000088421", "ratio": json.Number("1.5"), "list": []any{"a", nil, false}},
		"https://example.com/x": "free text",
	}
	if err := store.Save(ctx, partner, contract); err != nil {
		t.Fatal(err)
	}
	got, err := store.Source(partner).Fetch(ctx, corpus.DocumentKey{Namespace: ns, ID: "CAP-READ-ORDER-EXCEPTION"})
	if err != nil {
		t.Fatal(err)
	}
	if a, b := comparable(t, contract, ns), comparable(t, got, ns); a != b {
		t.Errorf("schema or extensions changed:\n want %s\n got  %s", a, b)
	}
	if db.count("json_schema_annotation") != 3 || db.count("json_schema_node") < 9 {
		t.Errorf("schema keywords were not decomposed: %d annotations, %d nodes", db.count("json_schema_annotation"), db.count("json_schema_node"))
	}
	if err := store.Save(ctx, partner, got); err != nil {
		t.Fatalf("the canonical schema is not a replay of its version: %v", err)
	}

	unit := edited(t, store, "OU-FINANCE", "extensions", map[string]any{"vendor:a": map[string]any{"deep": []any{json.Number("1"), "x"}}})
	if err := store.Save(ctx, partner, unit); err != nil {
		t.Fatal(err)
	}
	before := db.count("json_value")
	unit = edited(t, store, "OU-FINANCE", "extensions", map[string]any{"vendor:a": "replaced"}) // four nodes become one
	if err := store.Save(ctx, partner, unit); err != nil {
		t.Fatal(err)
	}
	if after := db.count("json_value"); after != before-3 {
		t.Errorf("an updated extension left its old value tree behind: %d nodes, then %d", before, after)
	}
}

// TestSchemaKeywordsTheTypedFormCannotHoldStayWhole checks that a keyword whose value a typed column cannot hold
// exactly is kept as an annotation, so even a schema the metaschema would refuse reads back unchanged.
func TestSchemaKeywordsTheTypedFormCannotHoldStayWhole(t *testing.T) {
	in := map[string]any{"minLength": "x", "maxItems": json.Number("1.5"), "items": []any{true}, "type": []any{"string", "string"},
		"required": []any{}, "enum": []any{}, "properties": map[string]any{"a": json.Number("1")}, "allOf": []any{},
		"format": strings.Repeat("f", 121), "minimum": "low", "uniqueItems": "yes", "dependentRequired": map[string]any{"a": []any{}}}
	node, err := decodeSchema(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(node.annotations) != len(in) || len(node.columns) != 0 || len(node.children) != 0 {
		t.Fatalf("typed %v, children %d, annotations %d", node.columns, len(node.children), len(node.annotations))
	}
	if canonicalJSON(node.encode()) != canonicalJSON(in) {
		t.Errorf("schema changed: %s", canonicalJSON(node.encode()))
	}
	if _, err := decodeSchema("not a schema"); !errors.Is(err, ErrUnrepresentable) {
		t.Errorf("a string decoded as a schema: %v", err)
	}
}

// TestEvidenceIsAppendedOnceAndKeepsEveryProperty covers the evidence store: duplicates, unknown properties, actors
// that are not their runtime's identity, and a structured record read back whole.
func TestEvidenceIsAppendedOnceAndKeepsEveryProperty(t *testing.T) {
	ctx := context.Background()
	store, db := newDocumentStore(t)
	ev := store.Evidence(partner)
	existing := harbor(t).OfKind(model.KindActionRecord)[0]
	if err := ev.Append(ctx, existing); !errors.Is(err, evidence.ErrDuplicate) {
		t.Errorf("duplicate evidence: %v", err)
	}
	extra := parseDocument(t, `{"charterSpecVersion":"1.8.0","namespace":"harbor.example","kind":"EvidenceRecord","id":"EVR-EXTRA",
		"enterpriseId":"ENT-HARBOR","category":"observed-fact","content":"x","recordedAt":"2026-06-18T17:00:00Z","bogus":1}`)
	if err := ev.Append(ctx, extra); !errors.Is(err, ErrUnmappedProperty) {
		t.Errorf("a property without a column was dropped: %v", err)
	}
	other := harbor(t).OfKind(model.KindAgentIdentity)
	var stranger string
	for _, a := range other {
		if a.ID != agent {
			stranger = a.ID
		}
	}
	impostor := relabel(t, existing, "ACT-IMPOSTOR")
	impostor.Value.(map[string]any)["actor"] = map[string]any{"kind": "AgentIdentity", "id": stranger}
	if err := ev.Append(ctx, impostor); !errors.Is(err, ErrRuntimeActor) {
		t.Errorf("an agent acted through another agent's runtime: %v", err)
	}
	record := parseDocument(t, `{"charterSpecVersion":"1.8.0","namespace":"harbor.example","kind":"EvidenceRecord","id":"EVR-STRUCTURED",
		"enterpriseId":"ENT-HARBOR","category":"agent-assertion","content":"assessed","recordedAt":"2026-06-18T17:00:00.123456+02:00",
		"sources":["prose source",{"locator":"https://example.com/a","mediaType":"text/plain","version":"1","digest":"sha256:`+strings.Repeat("a", 64)+`"}],
		"transformations":["t1","t2"],"integrity":{"method":"sha256"},
		"assessment":{"assessmentKind":"evaluation","subject":{"kind":"AgentDefinition","id":"AGENTDEF-ORDER-EXCEPTION-COORDINATOR-1"},
		"judge":{"kind":"Evaluator","id":"EVAL-1","external":true},"subjectDigest":"d","caseSet":{"id":"CS-1","version":"2"},
		"scores":[{"metric":"recall","value":0.875},{"metric":"accuracy","value":1}]},
		"extensions":{"vendor:n":null}}`)
	if err := ev.Append(ctx, record); err != nil {
		t.Fatal(err)
	}
	got, err := ev.Fetch(ctx, corpus.DocumentKey{Namespace: ns, ID: "EVR-STRUCTURED"})
	if err != nil {
		t.Fatal(err)
	}
	if a, b := comparable(t, record, ns), comparable(t, got, ns); a != b {
		t.Errorf("evidence changed:\n want %s\n got  %s", a, b)
	}
	if got.Value.(map[string]any)["sources"].([]any)[0] != "prose source" {
		t.Errorf("the order of sources changed: %v", got.Value.(map[string]any)["sources"])
	}
	tooFine := relabel(t, record, "EVR-NANO")
	tooFine.Value.(map[string]any)["recordedAt"] = "2026-06-18T17:00:00.123456789Z"
	if err := ev.Append(ctx, tooFine); !errors.Is(err, ErrUnrepresentable) {
		t.Errorf("a time finer than a microsecond was truncated: %v", err)
	}
	before := len(db.state.rows["evidence_record"])
	db.failQuery = qInsert("evidence_record_transformation")
	if err := ev.Append(ctx, relabel(t, record, "EVR-FAILS")); err == nil {
		t.Fatal("a failed insert was not reported")
	}
	db.failQuery = ""
	if len(db.state.rows["evidence_record"]) != before {
		t.Error("a failed append left rows behind")
	}
}

// relabel copies a document under another id.
func relabel(t *testing.T, d *model.Document, id string) *model.Document {
	t.Helper()
	value, err := documentValue(d)
	if err != nil {
		t.Fatal(err)
	}
	value["id"] = id
	out := &model.Document{Envelope: d.Envelope, Value: value}
	out.ID = id
	return out
}

func TestRelationalEvidenceStoreContract(t *testing.T) {
	ctx := context.Background()
	evidencetest.Run(t, func(t *testing.T) evidence.Store {
		store, _ := newRelationalStore(t)
		for _, namespace := range []string{"a.example", "b.example"} {
			docs := []string{
				`{"kind":"Enterprise","id":"ENT-1","name":"E"}`,
				`{"kind":"OrganizationUnit","id":"OU-1","name":"U","enterpriseId":"ENT-1"}`,
				`{"kind":"Responsibility","id":"RESP-1","name":"R","enterpriseId":"ENT-1","expectedOutcome":"done","accountable":{"kind":"OrganizationUnit","id":"OU-1"}}`,
				`{"kind":"CapabilityContract","id":"CAP-1","name":"C","enterpriseId":"ENT-1","purpose":"p","operationClass":"read","contractVersion":"1",
				  "inputs":[],"outputs":[],"preconditions":[],"outcomes":["done"],"businessErrors":[],"requiredAuthority":"r",
				  "constraints":{"approvalRequired":false,"sodConstraintIds":[],"limits":[]},"idempotency":{"mutating":false,"retrySemantics":"none"},
				  "evidenceRequirements":["e"]}`,
				`{"kind":"AgentIdentity","id":"AGENT-1","name":"A","enterpriseId":"ENT-1","lifecycleState":"active"}`,
				`{"kind":"AgentDefinition","id":"DEF-1","enterpriseId":"ENT-1","agentIdentityId":"AGENT-1","definitionVersion":"1","purpose":"p",
				  "accountable":{"kind":"OrganizationUnit","id":"OU-1"},"responsibilityIds":["RESP-1"],"triggers":["t"],"requiredInputs":["i"],
				  "expectedOutcomes":["o"],"capabilityIds":["CAP-1"],"policies":["p"],"escalationConditions":["e"],"confidenceBoundaries":["c"]}`,
				`{"kind":"AgentRuntime","id":"RT-1","name":"RT","lifecycleState":"active","enterpriseId":"ENT-1","agentIdentityId":"AGENT-1","agentDefinitionId":"DEF-1",
				  "definitionVersion":"1","operator":{"kind":"OrganizationUnit","id":"OU-1"}}`,
			}
			var batch []*model.Document
			for _, raw := range docs {
				batch = append(batch, parseDocument(t, `{"charterSpecVersion":"1.8.0","namespace":"`+namespace+`",`+raw[1:]))
			}
			if err := store.Save(ctx, partner, batch...); err != nil {
				t.Fatal(err)
			}
		}
		return store.Evidence(partner)
	}, func(s evidence.Store) evidence.Provider { return evidence.NewBaseProvider(s) })
}

// TestModulesInstallSelectively stores an organization in a database holding only the organization module and the
// modules it depends on, and refuses what needs any other module.
func TestModulesInstallSelectively(t *testing.T) {
	ctx := context.Background()
	modules := []string{"charter_json", "charter_object", "charter_identity", "charter_work", "charter_organization"}
	db := newMemoryDB(t, modules...)
	store, err := NewDocumentStore(ctx, db, modules...)
	if err != nil {
		t.Fatal(err)
	}
	var docs []*model.Document
	for _, raw := range []string{
		`"kind":"Enterprise","id":"ENT-1","name":"E"}`,
		`"kind":"OrganizationUnit","id":"OU-1","name":"U","enterpriseId":"ENT-1"}`,
		`"kind":"Position","id":"POS-1","name":"P","enterpriseId":"ENT-1","organizationUnitId":"OU-1"}`,
		`"kind":"HumanIdentity","id":"HUMAN-1","name":"H","enterpriseId":"ENT-1","lifecycleState":"active"}`,
		`"kind":"Responsibility","id":"RESP-1","name":"R","enterpriseId":"ENT-1","expectedOutcome":"o","accountable":{"kind":"Position","id":"POS-1"}}`,
		`"kind":"Assignment","id":"ASGN-1","name":"A","enterpriseId":"ENT-1","subject":{"kind":"HumanIdentity","id":"HUMAN-1"},
		 "target":{"kind":"Position","id":"POS-1"},"participation":"occupies","validity":{"from":"2026-01-01"}}`,
	} {
		docs = append(docs, parseDocument(t, `{"charterSpecVersion":"1.8.0","namespace":"org.example",`+raw))
	}
	if err := store.Save(ctx, partner, docs...); err != nil {
		t.Fatal(err)
	}
	for _, want := range docs {
		got, err := store.Source(partner).Fetch(ctx, corpus.DocumentKey{Namespace: want.Namespace, ID: want.ID})
		if err != nil || comparable(t, want, want.Namespace) != comparable(t, got, want.Namespace) {
			t.Errorf("%s: %v", want.ID, err)
		}
	}
	cited := relabel(t, docs[1], "OU-2")
	cited.Value.(map[string]any)["sourceEvidence"] = []any{"EVR-1"}
	if err := store.Save(ctx, partner, cited); !errors.Is(err, ErrModuleNotInstalled) || !strings.Contains(err.Error(), "charter_evidence") {
		t.Errorf("a property whose table another module holds: %v", err)
	}
	if _, err := store.Source(partner).List(ctx, model.KindTask); !errors.Is(err, ErrModuleNotInstalled) {
		t.Errorf("a kind of a module not installed was listed: %v", err)
	}
	if err := store.Evidence(partner).Append(ctx, harbor(t).OfKind(model.KindEvidenceRecord)[0]); !errors.Is(err, ErrModuleNotInstalled) {
		t.Errorf("evidence appended without the evidence module: %v", err)
	}
	for name, set := range map[string][]string{
		"missing dependency": {"charter_json", "charter_object", "charter_organization"},
		"unknown module":     {"charter_json", "charter_object", "charter_nothing"},
		"no object module":   {"charter_json"},
	} {
		if _, err := NewDocumentStore(ctx, db, set...); !errors.Is(err, ErrModuleNotInstalled) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
