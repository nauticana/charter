package keel

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/authority"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func TestDocumentStoreReadsEveryVersion(t *testing.T) {
	ctx := context.Background()
	store, _ := newDocumentStore(t)
	key := corpus.DocumentKey{Namespace: ns, ID: "TASK-OE-DETECT"}
	original := edited(t, store, key.ID, "name", nil).Name
	next := edited(t, store, key.ID, "name", "Detect faster")
	next.Value.(map[string]any)["taskDefinitionVersion"] = "2"
	if err := store.Save(ctx, partner, next); err != nil {
		t.Fatal(err)
	}
	versions, err := store.Versions(ctx, partner, key)
	if err != nil || len(versions) != 2 || versions[0].No != 1 || versions[0].Label != "1" || versions[1].Label != "2" {
		t.Fatalf("versions = %+v, %v", versions, err)
	}
	first, err := store.Source(partner).FetchVersion(ctx, key, "1")
	if err != nil || first.Name != original {
		t.Fatalf("version 1 = %+v, %v", first, err)
	}
	if current, err := store.Source(partner).Fetch(ctx, key); err != nil || current.Name != "Detect faster" {
		t.Fatalf("current = %+v, %v", current, err)
	}
	if pinned, err := store.Source(partner).FetchVersion(ctx, key, "1"); err != nil || pinned.Name != original {
		t.Fatalf("source version 1 = %+v, %v", pinned, err)
	}
	if current, err := store.Source(partner).FetchVersion(ctx, key, ""); err != nil || current.Name != "Detect faster" {
		t.Fatalf("source current = %+v, %v", current, err)
	}
	for name, err := range map[string]error{
		"unknown label": func() error { _, err := store.Source(partner).FetchVersion(ctx, key, "9"); return err }(),
		"unknown document": func() error {
			_, err := store.Versions(ctx, partner, corpus.DocumentKey{Namespace: ns, ID: "NONE"})
			return err
		}(),
		"other tenant": func() error { _, err := store.Source(partner+1).FetchVersion(ctx, key, "1"); return err }(),
	} {
		if !errors.Is(err, corpus.ErrNotFound) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := store.Versions(ctx, 0, key); !errors.Is(err, ErrNoTenant) {
		t.Errorf("no tenant: %v", err)
	}
}

func TestDocumentStoreNamesTheTableOfAKind(t *testing.T) {
	store, _ := newDocumentStore(t)
	for kind, want := range map[model.Kind]string{model.KindTask: "process_task", model.KindPosition: "position",
		model.KindBindingConformance: "system_binding_conformance"} {
		if got, ok := store.KindTable(kind); !ok || got != want {
			t.Errorf("%s: %s %t", kind, got, ok)
		}
	}
	if _, ok := store.KindTable("Unknown"); ok {
		t.Error("an unknown kind has a table")
	}
	partial, err := NewDocumentStore(context.Background(), newMemoryDB(t, "charter_json", "charter_object"), "charter_json", "charter_object")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := partial.KindTable(model.KindTask); ok {
		t.Error("a kind outside the installed modules has a table")
	}
}

func TestDocumentStorePagesAKind(t *testing.T) {
	ctx := context.Background()
	store, db := newDocumentStore(t)
	ids := func(q DocumentQuery) []string {
		t.Helper()
		docs, err := store.Page(ctx, partner, q)
		if err != nil {
			t.Fatal(err)
		}
		out := make([]string, len(docs))
		for i, d := range docs {
			out[i] = d.ID
		}
		return out
	}
	planning := &model.ObjectRef{Kind: model.KindOrganizationUnit, Namespace: ns, ID: "OU-PRODUCTION-PLANNING"}
	q := DocumentQuery{Kind: model.KindPosition, At: at, Parent: planning, Limit: 2}
	if got := ids(q); len(got) != 2 || got[0] != "PP-MGR-01" || got[1] != "PP-PLN-01" {
		t.Fatalf("first page = %v", got)
	}
	q.After = corpus.DocumentKey{Namespace: ns, ID: "PP-PLN-01"}
	if got := ids(q); len(got) != 1 || got[0] != "PP-PLN-02" {
		t.Fatalf("second page = %v", got)
	}
	if got := ids(DocumentQuery{Kind: model.KindPosition, At: at, Prefix: "credit", Limit: 10}); len(got) != 1 || got[0] != "POS-CREDIT-MANAGER" {
		t.Errorf("name prefix = %v", got)
	}
	if got := ids(DocumentQuery{Kind: model.KindPosition, At: at, Prefix: "pos-%", Limit: 10}); len(got) != 0 {
		t.Errorf("a LIKE wildcard in the prefix matched: %v", got)
	}
	if got := ids(DocumentQuery{Kind: model.KindPosition, At: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Limit: 10}); len(got) != 0 {
		t.Errorf("positions before their validity = %v", got)
	}
	process := &model.ObjectRef{Kind: model.KindBusinessProcess, Namespace: ns, ID: "PROC-RESOLVE-ORDER-EXCEPTION"}
	if got := ids(DocumentQuery{Kind: model.KindTask, At: at, Parent: process, Limit: 10}); len(got) != len(harbor(t).OfKind(model.KindTask)) {
		t.Errorf("tasks of the process = %v", got)
	}
	if db.calls[qObjectsOfKind] != 0 {
		t.Error("a page listed the whole kind")
	}
	if got, err := store.Page(ctx, partner+1, DocumentQuery{Kind: model.KindPosition, At: at, Limit: 10}); err != nil || len(got) != 0 {
		t.Errorf("another tenant's page = %d, %v", len(got), err)
	}
	for name, tc := range map[string]struct {
		q    DocumentQuery
		want error
	}{
		"no limit":          {DocumentQuery{Kind: model.KindPosition, At: at}, ErrInvalidQuery},
		"no time":           {DocumentQuery{Kind: model.KindPosition, Limit: 1}, ErrInvalidQuery},
		"kind of no parent": {DocumentQuery{Kind: model.KindRole, At: at, Parent: planning, Limit: 1}, ErrInvalidQuery},
		"unknown parent":    {DocumentQuery{Kind: model.KindPosition, At: at, Parent: &model.ObjectRef{Kind: model.KindOrganizationUnit, Namespace: ns, ID: "OU-NONE"}, Limit: 1}, corpus.ErrNotFound},
		"unmapped kind":     {DocumentQuery{Kind: "Unknown", At: at, Limit: 1}, ErrUnmappedProperty},
	} {
		if _, err := store.Page(ctx, partner, tc.q); !errors.Is(err, tc.want) {
			t.Errorf("%s: %v, want %v", name, err, tc.want)
		}
	}
}

func TestDocumentStorePagesConceptsByAnyLabel(t *testing.T) {
	ctx := context.Background()
	store, db := newDocumentStore(t)
	concept := parseDocument(t, `{"charterSpecVersion":"1.9.0","namespace":"harbor.example","kind":"WorkConcept","id":"WC-CREDIT-CONTROL",
		"name":"Credit Control","classificationId":"WORKCLASS-ONET-SOC-2019","externalId":"13-2041.01","preferredLabel":"Credit Controller",
		"labels":[{"language":"de","label":"Kreditsachbearbeiter"}]}`)
	if err := store.Save(ctx, partner, concept); err != nil {
		t.Fatal(err)
	}
	for prefix, want := range map[string]string{"kredit": "WC-CREDIT-CONTROL", "credit contr": "WC-CREDIT-CONTROL", "13-2041.01": "WC-CREDIT-CONTROL", "43-4151": "WC-ONET-ORDER-CLERKS"} {
		docs, err := store.Page(ctx, partner, DocumentQuery{Kind: model.KindWorkConcept, At: at, Prefix: prefix, Limit: 10})
		if err != nil || len(docs) != 1 || docs[0].ID != want {
			t.Errorf("prefix %q = %v, %v, want %s", prefix, docs, err, want)
		}
	}
	if docs, err := store.Page(ctx, partner+1, DocumentQuery{Kind: model.KindWorkConcept, At: at, Prefix: "kredit", Limit: 10}); err != nil || len(docs) != 0 {
		t.Errorf("another tenant's concepts = %v, %v", docs, err)
	}
	if db.calls[qPageOf(kindSpecs[model.KindWorkConcept], false)] == 0 || qPageOf(kindSpecs[model.KindWorkConcept], false) == qPage {
		t.Error("concepts were not paged by their own query")
	}
}

func TestPageQueriesBindEveryArgument(t *testing.T) {
	m, err := relational()
	if err != nil {
		t.Fatal(err)
	}
	in, err := m.install(nil)
	if err != nil {
		t.Fatal(err)
	}
	for kind, spec := range kindSpecs {
		args := 9 + len(spec.terms)
		if got := strings.Count(in.queries[qPageOf(spec, false)], "?"); got != args {
			t.Errorf("%s page binds %d arguments, Page passes %d", kind, got, args)
		}
		if got := strings.Count(in.queries[qPageOf(spec, true)], "?"); spec.parent != "" && got != args+1 {
			t.Errorf("%s page under a parent binds %d arguments, Page passes %d", kind, got, args+1)
		}
	}
}

func TestDocumentStoreReadsMeasurementsOfAPeriod(t *testing.T) {
	ctx := context.Background()
	store, db := newDocumentStore(t)
	measure := corpus.DocumentKey{Namespace: ns, ID: "OM-OE-RESOLUTION-TIME"}
	year := model.Validity{From: "2026-01-01", To: "2026-12-31"}
	records, err := store.Measurements(ctx, partner, year, measure)
	if err != nil || len(records) != 2 || records[0].ID != "EVR-OE-RESOLUTION-BASELINE" || records[1].ID != "EVR-OE-RESOLUTION-2026-Q2" ||
		db.calls[qMeasurements] != 1 || db.calls[qObjectsOfKind] != 0 {
		t.Fatalf("measurements = %d, %v", len(records), err)
	}
	if records, err := store.Measurements(ctx, partner, model.Validity{From: "2026-02-01", To: "2026-06-30"}, measure); err != nil || len(records) != 1 {
		t.Errorf("a window outside the period was kept: %d, %v", len(records), err)
	}
	if records, err := store.Measurements(ctx, partner, year); err != nil || records != nil {
		t.Errorf("no measures: %v, %v", records, err)
	}
	if _, err := store.Measurements(ctx, partner, model.Validity{From: "2026-01-01"}, measure); !errors.Is(err, ErrInvalidQuery) {
		t.Errorf("open period: %v", err)
	}
	if _, err := store.Measurements(ctx, partner, year, corpus.DocumentKey{Namespace: ns, ID: "OM-NONE"}); !errors.Is(err, corpus.ErrNotFound) {
		t.Errorf("unknown measure: %v", err)
	}
	if _, err := store.Measurements(ctx, partner+1, year, measure); !errors.Is(err, corpus.ErrNotFound) {
		t.Errorf("another tenant's measure: %v", err)
	}
}

func TestBindingConformancePinsTheBindingVersion(t *testing.T) {
	ctx := context.Background()
	store, db := newDocumentStore(t)
	const binding = "BIND-S4-RESERVE-ORDER-STOCK-1"
	evaluated := func(id, bindingID, version string) *model.Document {
		return parseDocument(t, `{"charterSpecVersion":"1.9.0","namespace":"harbor.example","kind":"BindingConformance","id":"`+id+`",
			"bindingId":"`+bindingID+`","bindingVersion":"`+version+`","evaluatedFeatures":[{"feature":"reservation-create","support":"supported"}],
			"conformanceProfile":"system-adapter","implementationVersion":"1.0","testedRuleIds":["CHR-RULE-BIND-001"],
			"verificationTypes":["runtime-behavioral"],"result":"conforming","resultDate":"2026-06-18"}`)
	}
	conformance := func(id, version string) *model.Document { return evaluated(id, binding, version) }
	if err := store.Save(ctx, partner, conformance("BC-1", "1")); err != nil {
		t.Fatal(err)
	}
	// A conformance may be saved with the version it pins, and saving it again is a replay.
	if err := store.Save(ctx, partner, conformance("BC-2", "2"), edited(t, store, binding, "bindingVersion", "2")); err != nil {
		t.Fatal(err)
	}
	rows := db.count("system_binding_conformance")
	if err := store.Save(ctx, partner, conformance("BC-2", "2")); err != nil || db.count("system_binding_conformance") != rows {
		t.Fatalf("replay: %v", err)
	}
	got, err := store.Source(partner).Fetch(ctx, corpus.DocumentKey{Namespace: ns, ID: "BC-1"})
	if err != nil {
		t.Fatal(err)
	}
	if c, err := corpus.Decode[model.BindingConformance](got); err != nil || c.BindingID.ID != binding || c.BindingVersion != "1" {
		t.Fatalf("conformance = %+v, %v", c, err)
	}
	if err := store.Save(ctx, partner, conformance("BC-9", "9")); !errors.Is(err, ErrUnknownVersion) {
		t.Errorf("a conformance pinned a version that is not stored: %v", err)
	}
	if err := store.Save(ctx, partner, evaluated("BC-OU", "OU-FINANCE", "1")); !errors.Is(err, ErrInvalidDocument) {
		t.Errorf("a conformance of a document that is not a binding: %v", err)
	}
}

func TestDocumentStoreKeepsEachVersionsDerivation(t *testing.T) {
	ctx := context.Background()
	store, _ := newDocumentStore(t)
	key := corpus.DocumentKey{Namespace: ns, ID: "TASK-OE-DETECT"}
	derived := func(label, digit string) *model.Document {
		d := edited(t, store, key.ID, "taskDefinitionVersion", label)
		d.Value.(map[string]any)["derivedFrom"] = map[string]any{
			"source":  map[string]any{"kind": "Task", "namespace": "reference.example", "id": key.ID},
			"version": label, "digest": "sha256:" + strings.Repeat(digit, 64), "adoption": "adopted",
		}
		return d
	}
	for _, label := range []string{"2", "3"} {
		if err := store.Save(ctx, partner, derived(label, label)); err != nil {
			t.Fatalf("version %s: %v", label, err)
		}
	}
	for _, label := range []string{"2", "3"} {
		d, err := store.Source(partner).FetchVersion(ctx, key, label)
		if err != nil || d.DerivedFrom == nil || d.DerivedFrom.Version != label || d.DerivedFrom.Digest != "sha256:"+strings.Repeat(label, 64) {
			t.Fatalf("version %s = %+v, %v", label, d, err)
		}
	}
	if err := store.Save(ctx, partner, derived("3", "4")); err == nil {
		t.Fatal("a stored version's derivation changed")
	}
}

func TestDocumentStoreReadsAnActorsGrants(t *testing.T) {
	ctx := context.Background()
	store, db := newDocumentStore(t)
	actor := model.ObjectRef{Kind: model.KindAgentIdentity, Namespace: ns, ID: "AGENT-ORDER-EXCEPTION-COORDINATOR"}
	at := time.Date(2026, 6, 18, 9, 0, 0, 0, time.UTC)
	grants, err := store.ActorGrants(ctx, partner, actor, at)
	if err != nil || len(grants) != 1 || grants[0].ID != "AUTH-OEC-STOCK-RESERVATION-2026" {
		t.Fatalf("grants in force = %+v, %v", grants, err)
	}
	g := grants[0]
	req := authority.Request{Namespace: ns, EnterpriseID: *g.EnterpriseID, Actor: model.ObjectRef{Kind: actor.Kind, ID: actor.ID},
		CapabilityID: g.CapabilityID, At: at}
	before := db.calls[qObjectsOfKind]
	if got, err := store.Grants(partner).Grants(ctx, req); err != nil || len(got) != 1 || db.calls[qObjectsOfKind] != before {
		t.Fatalf("grant source = %+v, %v; listed a kind %d times", got, err, db.calls[qObjectsOfKind]-before)
	}
	req.CapabilityID = model.Ref{ID: "CAP-OTHER"}
	if got, err := store.Grants(partner).Grants(ctx, req); err != nil || len(got) != 0 {
		t.Errorf("other capability = %+v, %v", got, err)
	}
	if got, err := store.ActorGrants(ctx, partner, model.ObjectRef{Kind: model.KindHumanIdentity, Namespace: ns, ID: "NONE"}, at); err != nil || len(got) != 0 {
		t.Errorf("unknown actor = %+v, %v", got, err)
	}

	if _, err := store.ActorGrants(ctx, partner, model.ObjectRef{Kind: actor.Kind, ID: actor.ID}, at); !errors.Is(err, ErrInvalidQuery) {
		t.Errorf("actor without a namespace: %v", err)
	}

	overdue := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	if got, err := store.ActorGrants(ctx, partner, actor, overdue); err != nil || len(got) != 1 || authority.Lapse(got[0], overdue) == "" {
		t.Fatalf("suspended grant = %+v, %v", got, err)
	}

	revoked := at.Add(time.Hour)
	if err := store.Save(ctx, partner, edited(t, store, g.ID, "revokedAt", revoked.Format(time.RFC3339))); err != nil {
		t.Fatal(err)
	}
	stored, err := store.ActorGrants(ctx, partner, actor, at)
	if err != nil || len(stored) != 1 || stored[0].RevokedAt == nil || !stored[0].RevokedAt.Equal(revoked) {
		t.Fatalf("revoked grant = %+v, %v", stored, err)
	}
	if got, err := store.ActorGrants(ctx, partner, actor, revoked); err != nil || len(got) != 0 {
		t.Errorf("grants held after revocation = %+v, %v", got, err)
	}
}

func TestDocumentStoreListsReferrers(t *testing.T) {
	ctx := context.Background()
	store, _ := newDocumentStore(t)
	capability := model.ObjectRef{Kind: model.KindCapabilityContract, Namespace: ns, ID: "CAP-RESERVE-ORDER-STOCK"}
	at := time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)
	refs, err := store.Referencing(ctx, partner, capability, at, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []Referrer{
		{Document: model.ObjectRef{Kind: model.KindAuthorityGrant, Namespace: ns, ID: "AUTH-OEC-STOCK-RESERVATION-2026"}, Property: "capabilityId"},
		{Document: model.ObjectRef{Kind: model.KindAgentDefinition, Namespace: ns, ID: "AGENTDEF-ORDER-EXCEPTION-COORDINATOR-1"}, Property: "capabilityIds"},
		{Document: model.ObjectRef{Kind: model.KindCapabilityBinding, Namespace: ns, ID: "BIND-S4-RESERVE-ORDER-STOCK-1"}, Property: "capabilityId"},
	} {
		if !slices.Contains(refs, want) {
			t.Errorf("missing %+v in %+v", want, refs)
		}
	}
	if !slices.IsSortedFunc(refs, func(a, b Referrer) int {
		return strings.Compare(a.Document.Namespace+" "+a.Document.ID+" "+a.Property, b.Document.Namespace+" "+b.Document.ID+" "+b.Property)
	}) {
		t.Errorf("unordered: %+v", refs)
	}
	if page, err := store.Referencing(ctx, partner, capability, at, 2); err != nil || !slices.Equal(page, refs[:2]) {
		t.Errorf("limited = %+v, %v", page, err)
	}
	if late, err := store.Referencing(ctx, partner, capability, time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC), 100); err != nil ||
		slices.ContainsFunc(late, func(r Referrer) bool { return r.Document.ID == "AUTH-OEC-STOCK-RESERVATION-2026" }) {
		t.Errorf("expired grant listed: %+v, %v", late, err)
	}
	enterprise := model.ObjectRef{Kind: model.KindEnterprise, Namespace: ns, ID: "ENT-HARBOR"}
	if owned, err := store.Referencing(ctx, partner, enterprise, at, 1); err != nil || len(owned) != 1 || owned[0].Property != "enterpriseId" {
		t.Errorf("documents of the enterprise = %+v, %v", owned, err)
	}
	for name, err := range map[string]error{
		"no limit": func() error { _, err := store.Referencing(ctx, partner, capability, at, 0); return err }(),
		"unknown": func() error {
			_, err := store.Referencing(ctx, partner, model.ObjectRef{Kind: capability.Kind, Namespace: ns, ID: "NONE"}, at, 1)
			return err
		}(),
	} {
		if err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}
