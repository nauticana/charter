package keel

import "github.com/nauticana/charter/sdk/model"

// fieldKind is how one document property maps to columns.
type fieldKind uint8

const (
	fText         fieldKind = iota // string
	fInt                           // integer
	fNumber                        // number, stored as NUMERIC
	fBool                          // boolean
	fFlag                          // boolean stored false when absent and written only when true
	fTime                          // date-time
	fDate                          // date
	fCode                          // enumeration word stored as its code
	fCodeOrText                    // a Charter-defined word in the code column, any other string in the text column
	fRef                           // idRef
	fObject                        // objectRef of any kind, or of an object outside the document set
	fArc                           // objectRef restricted to kinds, one column per kind
	fRefArc                        // idRef restricted to kinds, one column per kind resolved when written
	fPin                           // idRef and the version label it names, stored as the version key
	fJSON                          // any JSON value, stored as a json_value tree
	fSchema                        // an embedded JSON Schema, stored as a json_schema_node tree
	fConst                         // a constant the specification fixes, written on export only
	fApproved                      // approvedAction: a capability contract id or an action class
	fMeasured                      // a measurement value: minor units when its measure has a currency
	fMoney                         // minor units when the item has a currency, else a quantity
	fLimitValue                    // a limit value: minor units, number, text, or boolean
	fExponent                      // currencyExponent, which keel's currency table owns
	fActor                         // an action's actor: a human is stored, an agent is its runtime's identity
	fRuntimeAgent                  // an agent runtime's identity, which its definition fixes
)

// arc is one admitted kind of a restricted reference and the column holding it.
type arc struct {
	kind     model.Kind
	col      string
	external bool
}

type field struct {
	kind   fieldKind
	path   string
	col    string
	cols   []string
	codes  *codeList
	arcs   []arc
	label  string
	verCol string
	value  any
	fixed  bool
}

// columns lists the physical columns a field writes.
func (f field) columns() []string {
	switch f.kind {
	case fArc, fRefArc:
		out := make([]string, 0, len(f.arcs))
		for _, a := range f.arcs {
			out = append(out, a.col)
		}
		return out
	case fPin:
		return []string{f.col, f.verCol}
	case fMoney, fLimitValue, fCodeOrText:
		return f.cols
	case fApproved:
		return []string{"approved_capability_id", "approved_action_class"}
	case fMeasured:
		return []string{"measured_amount", "measured_quantity"}
	case fConst, fExponent, fRuntimeAgent:
		return nil
	}
	return []string{f.col}
}

// tableSpec maps an object, or the items of an array, to the rows of one table.
type tableSpec struct {
	name   string
	owner  []string // columns holding the parent's key, the owning document's id first
	order  string   // ordinal column of a sequence
	key    []string // natural key of a set, or of a sequence whose items children reference
	fields []field
	lists  []listSpec
	prose  string // column holding the string form of an item that may be a string or an object
	form   string // column set only in the object form
}

type listShape uint8

const (
	lArray listShape = iota
	lOne             // an optional nested object with lists of its own
	lMap             // an object whose members are rows keyed by member name
	lSplit           // a set of idRefs whose table depends on each member's kind
)

type splitTarget struct {
	kind  model.Kind
	table string
	col   string
}

type listSpec struct {
	path    string
	shape   listShape
	always  bool
	scalar  bool
	mapKey  string
	table   *tableSpec
	targets []splitTarget
}

// kindSpec maps one document kind; a versioned kind has an identity table and a version table as its root.
type kindSpec struct {
	kind     model.Kind
	code     string
	identity *tableSpec
	root     *tableSpec
	envelope *tableSpec
	label    string
	evidence bool
	once     bool
	objects  []string
	// actorRuntime names the agent actor and runtime columns of a record whose agent must be its runtime's identity.
	actorRuntime [2]string
}

func (k *kindSpec) versioned() bool { return k.identity != nil }

// tables visits every table spec of the kind, parents first.
func (k *kindSpec) tables(visit func(t *tableSpec, split *listSpec)) {
	var walk func(t *tableSpec)
	walk = func(t *tableSpec) {
		visit(t, nil)
		for i := range t.lists {
			l := &t.lists[i]
			if l.shape == lSplit {
				visit(l.table, l)
				continue
			}
			walk(l.table)
		}
	}
	walk(k.envelope)
	if k.identity != nil {
		walk(k.identity)
	}
	walk(k.root)
}

// Field constructors.

func text(path, col string) field    { return field{kind: fText, path: path, col: col} }
func integer(path, col string) field { return field{kind: fInt, path: path, col: col} }
func number(path, col string) field  { return field{kind: fNumber, path: path, col: col} }
func boolean(path, col string) field { return field{kind: fBool, path: path, col: col} }
func flag(path, col string) field    { return field{kind: fFlag, path: path, col: col} }
func stamp(path, col string) field   { return field{kind: fTime, path: path, col: col} }
func day(path, col string) field     { return field{kind: fDate, path: path, col: col} }
func ref(path, col string) field     { return field{kind: fRef, path: path, col: col} }
func object(path, col string) field  { return field{kind: fObject, path: path, col: col} }
func jsonOf(path, col string) field  { return field{kind: fJSON, path: path, col: col} }
func schemaOf(path, col string) field {
	return field{kind: fSchema, path: path, col: col}
}
func constant(path string, v any) field { return field{kind: fConst, path: path, value: v} }
func approved(path string) field        { return field{kind: fApproved, path: path} }
func measured(path string) field        { return field{kind: fMeasured, path: path} }
func exponent(path string) field        { return field{kind: fExponent, path: path} }
func runtimeAgent(path string) field    { return field{kind: fRuntimeAgent, path: path} }

func code(path, col string, c *codeList) field {
	return field{kind: fCode, path: path, col: col, codes: c}
}

func codeOrText(path, codeCol, textCol string, c *codeList) field {
	return field{kind: fCodeOrText, path: path, cols: []string{codeCol, textCol}, codes: c}
}

func arcOf(path string, arcs ...arc) field    { return field{kind: fArc, path: path, arcs: arcs} }
func refArcOf(path string, arcs ...arc) field { return field{kind: fRefArc, path: path, arcs: arcs} }
func on(kind model.Kind, col string) arc      { return arc{kind: kind, col: col} }
func outside(kind model.Kind, col string) arc { return arc{kind: kind, col: col, external: true} }

func pin(path, col, label, verCol string) field {
	return field{kind: fPin, path: path, col: col, label: label, verCol: verCol}
}

func money(path, amountCol, quantityCol string) field {
	return field{kind: fMoney, path: path, cols: []string{amountCol, quantityCol}}
}

func actor(path, humanCol string) field { return field{kind: fActor, path: path, col: humanCol} }

func (f field) once() field { f.fixed = true; return f }

// Table and list constructors.

func tbl(name string, owner ...string) *tableSpec { return &tableSpec{name: name, owner: owner} }

func (t *tableSpec) ordered(col string) *tableSpec   { t.order = col; return t }
func (t *tableSpec) keyed(cols ...string) *tableSpec { t.key = cols; return t }
func (t *tableSpec) with(fields ...field) *tableSpec {
	t.fields = append(t.fields, fields...)
	return t
}
func (t *tableSpec) sub(lists ...listSpec) *tableSpec { t.lists = append(t.lists, lists...); return t }
func (t *tableSpec) proseOr(prose, form string) *tableSpec {
	t.prose, t.form = prose, form
	return t
}

func objects(path string, t *tableSpec) listSpec { return listSpec{path: path, table: t} }
func one(path string, t *tableSpec) listSpec     { return listSpec{path: path, shape: lOne, table: t} }
func scalars(path string, t *tableSpec, f field) listSpec {
	f.path = ""
	t.fields = append(t.fields, f)
	return listSpec{path: path, scalar: true, table: t}
}
func (l listSpec) required() listSpec { l.always = true; return l }

// refSet is a set of idRefs in a child table keyed by the referenced object.
func refSet(path, table, owner, col string) listSpec {
	return scalars(path, tbl(table, owner).keyed(col), ref("", col))
}

// codeSet is a set of enumeration words in a child table keyed by the code.
func codeSet(path, table string, owner []string, col string, c *codeList) listSpec {
	return scalars(path, tbl(table, owner...).keyed(col), code("", col, c))
}

// statements is an ordered list of strings in a child table.
func statements(path, table string, owner []string, col string) listSpec {
	return scalars(path, tbl(table, owner...).ordered("ordinal"), text("", col))
}

// split is a set of idRefs whose members land in one table per admitted kind.
func split(path, owner string, targets ...splitTarget) listSpec {
	return listSpec{path: path, shape: lSplit, table: tbl("#split:"+path, owner).keyed(colSplitRef), targets: targets}
}

func into(kind model.Kind, table, col string) splitTarget {
	return splitTarget{kind: kind, table: table, col: col}
}

// Shared tables keyed by the document version.

var versionOwner = []string{"enterprise_object_id", "version_no"}

func sourceEvidence() listSpec {
	return scalars("sourceEvidence", tbl("enterprise_object_source_evidence", versionOwner...).keyed("evidence_record_id"),
		ref("", "evidence_record_id"))
}

func externalIdentifiers() listSpec {
	return objects("externalIdentifiers", tbl("enterprise_object_external_identifier", versionOwner...).
		keyed("system_profile_id", "external_id").
		with(ref("systemProfileId", "system_profile_id"), text("externalId", "external_id"),
			day("validity.from", "valid_from"), day("validity.to", "valid_to")))
}

func limits(path string) listSpec {
	return objects(path, tbl("enterprise_object_limit", versionOwner...).ordered("ordinal").
		with(text("limitKind", "limit_kind"), code("operator", "operator", operatorCodes), text("unit", "unit"),
			text("currency", "currency_code"), exponent("currencyExponent"),
			field{kind: fLimitValue, path: "value", cols: []string{"amount", "number_value", "text_value", "boolean_value"}}))
}

func featureSupport(path string) listSpec {
	return objects(path, tbl("binding_feature_support", versionOwner...).keyed("feature").
		with(text("feature", "feature"), code("support", "support", supportCodes), text("notes", "notes"))).required()
}

func evidenceRequirements() listSpec {
	return objects("evidenceRequirements", tbl("evidence_requirement", versionOwner...).ordered("ordinal").
		proseOr("statement", "information_definition_id").
		with(ref("informationId", "information_definition_id"), text("attribute", "attribute"),
			boolean("required", "required"), integer("maxAgeSeconds", "max_age_seconds")))
}

func lifecycleHistory() listSpec {
	return objects("lifecycleHistory", tbl("identity_lifecycle_transition", versionOwner...).keyed("effective_at").
		with(code("state", "lifecycle_state", identityLifecycleCodes), stamp("effectiveAt", "effective_at"),
			object("authority", "authority_id")))
}

func testedRules() listSpec {
	return scalars("testedRuleIds", tbl("conformance_tested_rule", versionOwner...).keyed("rule_id"), text("", "rule_id")).required()
}

func verificationTypes() listSpec {
	return codeSet("verificationTypes", "conformance_verification_type", versionOwner, "verification_type", verificationCodes).required()
}

// envelopeOf maps the properties every document carries; label is the version property of a versioned kind.
func envelopeOf(label string) *tableSpec {
	t := tbl("enterprise_object_version", versionOwner...).with(
		text("charterSpecVersion", "spec_version"), text("name", "name"), text("description", "description"),
		code("lifecycleState", "lifecycle_state", lifecycleCodes), day("validity.from", "valid_from"), day("validity.to", "valid_to"))
	if label != "" {
		t.fields = append(t.fields, text(label, "definition_version"))
	}
	return t.sub(
		one("derivedFrom", tbl("enterprise_object_provenance", versionOwner...).with(
			text("source.kind", "source_kind"), text("source.namespace", "source_namespace"), text("source.id", "source_identifier"),
			flag("source.external", "source_external"), text("version", "source_version"), text("digest", "source_digest"),
			code("adoption", "adoption", adoptionCodes))),
		listSpec{path: "extensions", shape: lMap, mapKey: "extension_name",
			table: tbl("enterprise_object_extension", versionOwner...).keyed("extension_name").with(jsonOf("", "json_value_id"))},
	)
}
