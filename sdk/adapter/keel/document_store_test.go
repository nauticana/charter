package keel

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/identity"
	"github.com/nauticana/charter/sdk/model"
)

const partner = int64(7)

// newDocumentStore returns a store holding every Harbor document, evidence included, for the test tenant.
func newDocumentStore(t *testing.T) (*DocumentStore, *memoryDB) {
	t.Helper()
	store, db := newRelationalStore(t)
	storeAll(t, store, partner, harbor(t).Documents())
	return store, db
}

// edited returns a copy of the stored document with one top-level property replaced.
func edited(t *testing.T, store *DocumentStore, id, property string, value any) *model.Document {
	t.Helper()
	d, err := store.Source(partner).Fetch(context.Background(), corpus.DocumentKey{Namespace: ns, ID: id})
	if err != nil {
		t.Fatal(err)
	}
	v := map[string]any{}
	maps.Copy(v, d.Value.(map[string]any))
	v[property] = value
	return &model.Document{Envelope: d.Envelope, Value: v}
}

func TestDocumentStoreServesTheTenantsCurrentDocuments(t *testing.T) {
	ctx := context.Background()
	store, db := newDocumentStore(t)
	source := store.Source(partner)
	tasks, err := source.List(ctx, model.KindTask)
	if err != nil || len(tasks) != len(harbor(t).OfKind(model.KindTask)) || tasks[0].Kind != model.KindTask {
		t.Fatalf("tasks = %d, %v", len(tasks), err)
	}
	actor, err := identity.NewBaseResolver(source).ActiveAt(ctx, ns, model.ObjectRef{Kind: model.KindAgentIdentity, ID: agent}, at)
	if err != nil || actor.ID != agent {
		t.Fatalf("providers resolve over the store: %+v %v", actor, err)
	}
	if _, err := store.Source(partner+1).Fetch(ctx, corpus.DocumentKey{Namespace: ns, ID: agent}); !errors.Is(err, corpus.ErrNotFound) {
		t.Errorf("another tenant's document is visible: %v", err)
	}
	if none, err := store.Source(partner+1).List(ctx, model.KindTask); err != nil || len(none) != 0 {
		t.Errorf("another tenant's documents are listed: %d %v", len(none), err)
	}
	if _, err := store.Source(0).List(ctx, model.KindTask); !errors.Is(err, ErrNoTenant) {
		t.Errorf("a source without a tenant answered: %v", err)
	}
	if err := store.Save(ctx, 0, tasks[0]); !errors.Is(err, ErrNoTenant) {
		t.Errorf("save without a tenant: %v", err)
	}
	if _, err := NewDocumentStore(ctx, nil); !errors.Is(err, ErrNoDatabase) {
		t.Errorf("store without a database: %v", err)
	}
	var stream any
	for _, o := range db.state.rows["enterprise_object"] {
		if o["kind"] == kindCodes.codes[string(model.KindValueStream)] {
			stream = o["id"]
		}
	}
	db.state.rows["enterprise_object_version"] = slices.DeleteFunc(db.state.rows["enterprise_object_version"], func(r map[string]any) bool {
		return r["enterprise_object_id"] == stream
	})
	if _, err := source.List(ctx, model.KindValueStream); !errors.Is(err, ErrDocumentIncomplete) {
		t.Errorf("a document whose rows are missing was served: %v", err)
	}
}

func TestDocumentStoreNeverEditsAStoredDefinitionVersion(t *testing.T) {
	ctx := context.Background()
	store, db := newDocumentStore(t)
	versions := db.count("process_task_version")
	if err := store.Save(ctx, partner, edited(t, store, "TASK-OE-DETECT", "name", "Detect faster")); !errors.Is(err, ErrPublishedVersion) {
		t.Fatalf("changed task under its stored version: %v", err)
	}
	if db.count("process_task_version") != versions || db.rollbacks == 0 {
		t.Fatal("a refused write left a version behind or was not rolled back")
	}
	next := edited(t, store, "TASK-OE-DETECT", "name", "Detect faster")
	next.Value.(map[string]any)["taskDefinitionVersion"] = "2"
	if err := store.Save(ctx, partner, next); err != nil {
		t.Fatal(err)
	}
	current, err := store.Source(partner).Fetch(ctx, corpus.DocumentKey{Namespace: ns, ID: "TASK-OE-DETECT"})
	if err != nil || current.Name != "Detect faster" || db.count("process_task_version") != versions+1 {
		t.Fatalf("new version is not current: %+v %v", current, err)
	}
	restored := edited(t, store, "TASK-OE-DETECT", "taskDefinitionVersion", "1")
	if err := store.Save(ctx, partner, restored); !errors.Is(err, ErrPublishedVersion) {
		t.Errorf("a version already stored was reused: %v", err)
	}

	units := db.count("organization_unit")
	unit := edited(t, store, "OU-FINANCE", "name", "Finance and Treasury")
	if err := store.Save(ctx, partner, unit); err != nil || db.count("organization_unit") != units {
		t.Fatalf("an unversioned kind is updated in place: %v", err)
	}
	if got, err := store.Source(partner).Fetch(ctx, corpus.DocumentKey{Namespace: ns, ID: "OU-FINANCE"}); err != nil || got.Name != "Finance and Treasury" {
		t.Fatalf("updated unit: %+v %v", got, err)
	}
}

func TestDocumentStoreKeepsFixedFacts(t *testing.T) {
	ctx := context.Background()
	store, _ := newDocumentStore(t)
	const definition = "AGENTDEF-ORDER-EXCEPTION-COORDINATOR-1"
	other := edited(t, store, definition, "definitionVersion", "2")
	other.Value.(map[string]any)["agentIdentityId"] = "AGENT-PLANNING-DATA-QUALITY"
	if err := store.Save(ctx, partner, other); !errors.Is(err, ErrFixedChanged) {
		t.Errorf("a definition changed its agent identity: %v", err)
	}
	if err := store.Save(ctx, partner, edited(t, store, definition, "definitionVersion", "2")); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, partner, edited(t, store, "RT-OEC-PROD-01", "definitionVersion", "2")); !errors.Is(err, ErrFixedChanged) {
		t.Errorf("a runtime changed its definition: %v", err)
	}
	if err := store.Save(ctx, partner, relabel(t, edited(t, store, "RT-OEC-PROD-01", "definitionVersion", "9"), "RT-NEW")); !errors.Is(err, ErrUnknownVersion) {
		t.Errorf("a runtime pinned a version that is not stored: %v", err)
	}
	decision := edited(t, store, "MEDDEC-OE-0042-01", "stage", "retrieval")
	if err := store.Save(ctx, partner, decision); !errors.Is(err, ErrRecordWritten) {
		t.Errorf("a mediation decision was rewritten: %v", err)
	}
	measure := edited(t, store, "OM-OE-RESOLUTION-TIME", "currency", "JPY")
	measure.Value.(map[string]any)["currencyExponent"] = 2
	delete(measure.Value.(map[string]any), "unit")
	if err := store.Save(ctx, partner, measure); !errors.Is(err, ErrCurrencyExponent) {
		t.Errorf("a currency exponent other than keel's was stored: %v", err)
	}
}

func TestApprovedActionIsACapabilityOrAClass(t *testing.T) {
	ctx := context.Background()
	store, db := newDocumentStore(t)
	approval := db.state.rows["authority_approval"][0]
	if approval["approved_capability_id"] == nil || approval["approved_action_class"] != nil {
		t.Fatalf("an approval naming a stored capability stored it as a class: %v", approval)
	}
	class := edited(t, store, "APPR-OE-2026-0042-CREDIT-01", "approvedAction", "manual-credit-override")
	if err := store.Save(ctx, partner, class); err != nil {
		t.Fatal(err)
	}
	approval = db.state.rows["authority_approval"][0]
	if approval["approved_capability_id"] != nil || approval["approved_action_class"] != "manual-credit-override" {
		t.Fatalf("an action class was stored as %v", approval)
	}
	got, err := store.Source(partner).Fetch(ctx, corpus.DocumentKey{Namespace: ns, ID: "APPR-OE-2026-0042-CREDIT-01"})
	if err != nil || got.Value.(map[string]any)["approvedAction"] != "manual-credit-override" {
		t.Fatalf("approved action read back: %v %v", got, err)
	}
}

func TestDocumentStoreRefusesInvalidWrites(t *testing.T) {
	ctx := context.Background()
	store, db := newDocumentStore(t)
	objects := db.count("enterprise_object")

	dangling := edited(t, store, "POS-CREDIT-MANAGER", "organizationUnitId", "OU-NONE")
	dangling.ID, dangling.Value.(map[string]any)["id"] = "POS-NEW", "POS-NEW"
	var invalid *InvalidDocumentsError
	if err := store.Save(ctx, partner, dangling); !errors.As(err, &invalid) || !errors.Is(err, ErrInvalidDocument) || invalid.Findings[0].DocumentID != "POS-NEW" {
		t.Fatalf("dangling reference: %v", err)
	}
	if err := store.Save(ctx, partner+1, edited(t, store, "POS-CREDIT-MANAGER", "name", "x")); !errors.As(err, &invalid) {
		t.Fatalf("references resolve only within the writing tenant: %v", err)
	}

	role := edited(t, store, "POS-CREDIT-MANAGER", "kind", string(model.KindRole))
	role.Kind = model.KindRole
	for _, k := range []string{"organizationUnitId", "positionTypeId", "validity", "externalIdentifiers"} {
		delete(role.Value.(map[string]any), k)
	}
	if err := store.Save(ctx, partner, role); !errors.Is(err, ErrKindChanged) {
		t.Fatalf("a stored document changed kind: %v", err)
	}
	otherEnterprise := edited(t, store, "OU-CREDIT-CONTROL", "enterpriseId", "ENT-OTHER")
	delete(otherEnterprise.Value.(map[string]any), "parentUnitId")
	enterprise := parseDocument(t, `{"charterSpecVersion":"1.8.0","namespace":"harbor.example","kind":"Enterprise","id":"ENT-OTHER","name":"Other"}`)
	if err := store.Save(ctx, partner, otherEnterprise, enterprise); !errors.Is(err, ErrEnterpriseChanged) {
		t.Fatalf("a stored document changed enterprise and invalidated inbound references: %v", err)
	}

	action := harbor(t).OfKind(model.KindActionRecord)[0]
	if err := store.Save(ctx, partner, action); !errors.Is(err, ErrEvidenceDocument) {
		t.Fatalf("evidence saved as a document: %v", err)
	}
	if err := store.Evidence(partner).Append(ctx, edited(t, store, "OU-FINANCE", "name", "x")); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("a document appended as evidence: %v", err)
	}
	if err := store.Save(ctx, partner, nil); err == nil {
		t.Fatal("nil document saved")
	}

	db.failCommit = true
	if err := store.Save(ctx, partner, edited(t, store, "OU-FINANCE", "name", "Group Finance")); err == nil {
		t.Fatal("commit failure was not reported")
	}
	db.failCommit = false
	db.failQuery = qUpdate("organization_unit")
	if err := store.Save(ctx, partner, edited(t, store, "OU-FINANCE", "name", "Group Finance")); err == nil {
		t.Fatal("update failure was not reported")
	}
	db.failQuery = ""
	if db.count("enterprise_object") != objects {
		t.Fatal("a refused or failed write changed the store")
	}
	if got, _ := store.Source(partner).Fetch(ctx, corpus.DocumentKey{Namespace: ns, ID: "OU-FINANCE"}); got.Name == "Group Finance" {
		t.Fatal("a failed write was committed")
	}
}

func TestDocumentStoreWriteReadsOnlyReferencedDocuments(t *testing.T) {
	ctx := context.Background()
	store, db := newDocumentStore(t)
	position := edited(t, store, "POS-CREDIT-MANAGER", "name", "Head of Credit")
	clear(db.calls)
	if err := store.Save(ctx, partner, position); err != nil {
		t.Fatal(err)
	}
	if db.calls[qObjectsOfKind] != 0 {
		t.Fatal("a write listed stored documents")
	}
	if got := db.calls[qObjectByName]; got > 10 {
		t.Fatalf("a one-document write looked up %d objects of %d stored", got, db.count("enterprise_object"))
	}

	unit := edited(t, store, "OU-FINANCE", "name", "x")
	db.failQuery = qSelectBy("organization_unit", "id")
	var invalid *InvalidDocumentsError
	if err := store.Save(ctx, partner, unit); err == nil || errors.As(err, &invalid) {
		t.Fatalf("an unreadable reference must fail the write as an error, not a finding: %v", err)
	}
}
