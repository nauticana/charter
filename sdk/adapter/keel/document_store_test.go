package keel

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/identity"
	"github.com/nauticana/charter/sdk/model"
)

const partner = int64(7)

// harborDocuments are the Harbor documents a document store keeps: everything except appended evidence.
func harborDocuments(t *testing.T) []*model.Document {
	var out []*model.Document
	for _, d := range harbor(t).Documents() {
		if !slices.Contains(evidenceKinds, d.Kind) {
			out = append(out, d)
		}
	}
	return out
}

func newDocumentStore(t *testing.T) (*DocumentStore, *memoryDB) {
	t.Helper()
	db := newMemoryDB()
	store, err := NewDocumentStore(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), partner, harborDocuments(t)...); err != nil {
		t.Fatal(err)
	}
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
	for k, x := range d.Value.(map[string]any) {
		v[k] = x
	}
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
	if _, err := store.Source(0).List(ctx, model.KindTask); !errors.Is(err, ErrNoTenant) {
		t.Errorf("a source without a tenant answered: %v", err)
	}
	if err := store.Save(ctx, 0, tasks[0]); !errors.Is(err, ErrNoTenant) {
		t.Errorf("save without a tenant: %v", err)
	}
	k := docKey{partner, ns, "TASK-OE-DETECT"}
	db.state.revisions[k][0].content = `{"charterSpecVersion":"1.2.0","namespace":"harbor.example","id":"TASK-OE-DETECT","kind":"Task"}`
	if _, err := source.Fetch(ctx, corpus.DocumentKey{Namespace: ns, ID: "TASK-OE-DETECT"}); !errors.Is(err, ErrStoredDocument) {
		t.Errorf("content that no longer matches its digest was served: %v", err)
	}
	if _, err := NewDocumentStore(ctx, nil); !errors.Is(err, ErrNoDatabase) {
		t.Errorf("store without a database: %v", err)
	}
}

func TestDocumentStoreNeverEditsAStoredDefinitionVersion(t *testing.T) {
	ctx := context.Background()
	store, db := newDocumentStore(t)
	key := docKey{partner, ns, "TASK-OE-DETECT"}
	if err := store.Save(ctx, partner, harborDocuments(t)...); err != nil || len(db.state.revisions[key]) != 1 {
		t.Fatalf("an identical replay appended a revision: %v, %d", err, len(db.state.revisions[key]))
	}
	if err := store.Save(ctx, partner, edited(t, store, "TASK-OE-DETECT", "name", "Detect faster")); !errors.Is(err, ErrPublishedVersion) {
		t.Fatalf("changed task under its stored version: %v", err)
	}
	if len(db.state.revisions[key]) != 1 || db.rollbacks == 0 {
		t.Fatal("a refused write left a revision behind or was not rolled back")
	}
	next := edited(t, store, "TASK-OE-DETECT", "name", "Detect faster")
	next.Value.(map[string]any)["taskDefinitionVersion"] = "2"
	if err := store.Save(ctx, partner, next); err != nil {
		t.Fatal(err)
	}
	current, err := store.Source(partner).Fetch(ctx, corpus.DocumentKey{Namespace: ns, ID: "TASK-OE-DETECT"})
	if err != nil || current.Name != "Detect faster" || len(db.state.revisions[key]) != 2 {
		t.Fatalf("new version is not current: %+v %v", current, err)
	}
	restored := edited(t, store, "TASK-OE-DETECT", "taskDefinitionVersion", "1")
	if err := store.Save(ctx, partner, restored); !errors.Is(err, ErrPublishedVersion) {
		t.Errorf("a version already stored was reused: %v", err)
	}

	unit := edited(t, store, "OU-FINANCE", "name", "Finance and Treasury")
	if err := store.Save(ctx, partner, unit); err != nil || len(db.state.revisions[docKey{partner, ns, "OU-FINANCE"}]) != 2 {
		t.Fatalf("an unversioned kind appends a revision: %v", err)
	}
}

func TestDocumentStoreRefusesInvalidWrites(t *testing.T) {
	ctx := context.Background()
	store, db := newDocumentStore(t)
	before := len(db.state.kinds)

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
	for _, k := range []string{"organizationUnitId", "positionTypeId", "validity"} {
		delete(role.Value.(map[string]any), k)
	}
	if err := store.Save(ctx, partner, role); !errors.Is(err, ErrKindChanged) {
		t.Fatalf("a stored document changed kind: %v", err)
	}
	otherEnterprise := edited(t, store, "OU-CREDIT-CONTROL", "enterpriseId", "ENT-OTHER")
	delete(otherEnterprise.Value.(map[string]any), "parentUnitId")
	enterprise, err := (corpus.Parser{}).Parse([]byte(`{"charterSpecVersion":"1.2.0","namespace":"harbor.example","kind":"Enterprise","id":"ENT-OTHER","name":"Other"}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, partner, otherEnterprise, enterprise); !errors.Is(err, ErrEnterpriseChanged) {
		t.Fatalf("a stored document changed enterprise and invalidated inbound references: %v", err)
	}

	evidence := harbor(t).OfKind(model.KindActionRecord)[0]
	if err := store.Save(ctx, partner, evidence); !errors.Is(err, ErrEvidenceDocument) {
		t.Fatalf("evidence saved as a document: %v", err)
	}
	if err := store.Save(ctx, partner, nil); err == nil {
		t.Fatal("nil document saved")
	}

	db.failCommit = true
	if err := store.Save(ctx, partner, edited(t, store, "OU-FINANCE", "name", "Group Finance")); err == nil {
		t.Fatal("commit failure was not reported")
	}
	db.failCommit = false
	db.failQuery = qRevisionInsert
	if err := store.Save(ctx, partner, edited(t, store, "OU-FINANCE", "name", "Group Finance")); err == nil {
		t.Fatal("insert failure was not reported")
	}
	if len(db.state.kinds) != before || len(db.state.revisions[docKey{partner, ns, "OU-FINANCE"}]) != 1 {
		t.Fatal("a refused or failed write changed the store")
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
	if got := db.calls[qDocumentCurrent]; got != 3 {
		t.Fatalf("a one-document write fetched %d documents, want its 3 references of %d stored", got, len(db.state.kinds))
	}
	if db.calls[qDocumentsOfKind] != 0 {
		t.Fatal("a write listed stored documents")
	}

	unit := edited(t, store, "OU-FINANCE", "name", "x")
	db.failQuery = qDocumentCurrent
	var invalid *InvalidDocumentsError
	if err := store.Save(ctx, partner, unit); err == nil || errors.As(err, &invalid) {
		t.Fatalf("an unreadable reference must fail the write as an error, not a finding: %v", err)
	}
}
