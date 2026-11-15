package keel

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/nauticana/keel/common"
	"github.com/nauticana/keel/port"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/validate"
)

var (
	ErrNoDatabase        = errors.New("no keel database")
	ErrNoTenant          = errors.New("no keel tenant")
	ErrEvidenceDocument  = errors.New("evidence is appended through an evidence store, not saved as a document")
	ErrKindChanged       = errors.New("a stored document keeps its kind")
	ErrEnterpriseChanged = errors.New("a stored document keeps its enterprise")
	ErrPublishedVersion  = errors.New("a stored definition version cannot change")
	ErrInvalidDocument   = errors.New("invalid documents")
	ErrInvalidQuery      = errors.New("invalid document query")
)

// InvalidDocumentsError carries the findings that kept a write from being committed.
type InvalidDocumentsError struct {
	Findings []validate.Finding
}

func (e *InvalidDocumentsError) Error() string {
	messages := make([]string, 0, len(e.Findings))
	for _, f := range e.Findings {
		messages = append(messages, fmt.Sprintf("%s %s: %s", f.DocumentID, f.RuleID, f.Message))
	}
	return ErrInvalidDocument.Error() + ": " + strings.Join(messages, "; ")
}

func (e *InvalidDocumentsError) Unwrap() error { return ErrInvalidDocument }

// evidenceKinds are appended once through Evidence and never saved as documents.
var evidenceKinds = []model.Kind{model.KindActionRecord, model.KindEvidenceRecord, model.KindEvidenceBundle, model.KindExceptionRecord, model.KindEscalation}

// DocumentStore keeps a tenant's documents in the sdk/schema tables, object references enforced by foreign keys. A
// document keeps its kind and enterprise. A versioned definition stores each declared version once, so a changed
// definition must declare a new version (CHR-PROC-005, CHR-AGENT-006); other documents are updated in place. Evidence
// is appended once and never changed. Saves are validated for structure and references before commit, fetching only
// the documents they reference.
//
// A store serves the kinds of the schema modules the database installs: modules names the groups of
// sdk/schema/dependency.yml passed to keel's schemagen, every module when none is named. A kind outside them, or a
// property whose table they lack, is refused with ErrModuleNotInstalled.
type DocumentStore struct {
	db        port.DatabaseRepository
	queries   port.QueryService
	model     *installation
	additions *validate.Additions
}

func NewDocumentStore(ctx context.Context, db port.DatabaseRepository, modules ...string) (*DocumentStore, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	full, err := relational()
	if err != nil {
		return nil, err
	}
	m, err := full.install(modules)
	if err != nil {
		return nil, err
	}
	additions, err := validate.NewAdditions()
	if err != nil {
		return nil, err
	}
	queries := db.GetQueryService(ctx, m.queries)
	if queries == nil {
		return nil, fmt.Errorf("%w: no query service", ErrNoDatabase)
	}
	return &DocumentStore{db: db, queries: queries, model: m, additions: additions}, nil
}

// Source returns one tenant's current documents, evidence included, for Charter providers.
func (s *DocumentStore) Source(partnerID int64) corpus.Source {
	return tenantStore{store: s, partnerID: partnerID}
}

// ObjectIDs returns, in order, the enterprise_object ids of a tenant's stored documents, the keys downstream foreign
// keys name, in one query. Each reference names its kind and namespace; one not stored as that kind is
// corpus.ErrNotFound.
func (s *DocumentStore) ObjectIDs(ctx context.Context, partnerID int64, refs ...model.ObjectRef) ([]int64, error) {
	r, err := tenantStore{store: s, partnerID: partnerID}.reader()
	if err != nil || len(refs) == 0 {
		return nil, err
	}
	namespaces, identifiers := make([]string, len(refs)), make([]string, len(refs))
	for i, ref := range refs {
		namespaces[i], identifiers[i] = ref.Namespace, ref.ID
	}
	objs, err := r.objectRows(ctx, qObjectsByName, namespaces, identifiers)
	if err != nil {
		return nil, fmt.Errorf("charter documents: %w", err)
	}
	stored := make(map[model.ObjectRef]int64, len(objs))
	for _, o := range objs {
		if o.code != externalKindCode {
			stored[model.ObjectRef{Kind: model.Kind(kindCodes.words[o.code]), Namespace: o.key.NS, ID: o.key.ID}] = o.id
		}
	}
	out := make([]int64, len(refs))
	for i, ref := range refs {
		id, found := stored[model.ObjectRef{Kind: ref.Kind, Namespace: ref.Namespace, ID: ref.ID}]
		if !found {
			return nil, fmt.Errorf("%w: %s %s:%s", corpus.ErrNotFound, ref.Kind, ref.Namespace, ref.ID)
		}
		out[i] = id
	}
	return out, nil
}

// ObjectRefs returns, in order, the documents enterprise_object ids store, each with its kind and namespace; an
// external object or an unknown id is corpus.ErrNotFound.
func (s *DocumentStore) ObjectRefs(ctx context.Context, partnerID int64, ids ...int64) ([]model.ObjectRef, error) {
	r, err := tenantStore{store: s, partnerID: partnerID}.reader()
	if err != nil {
		return nil, err
	}
	objs, err := r.objectsByID(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("charter objects: %w", err)
	}
	out := make([]model.ObjectRef, len(ids))
	for i, id := range ids {
		o, found := objs[id]
		if !found || o.code == externalKindCode {
			return nil, fmt.Errorf("%w: object %d", corpus.ErrNotFound, id)
		}
		out[i] = model.ObjectRef{Kind: model.Kind(kindCodes.words[o.code]), Namespace: o.key.NS, ID: o.key.ID}
	}
	return out, nil
}

// KindTable returns the table holding a kind's documents, the identity table of a versioned kind, the table keel's
// TABLE grants name; ok is false for a kind the installed modules do not hold.
func (s *DocumentStore) KindTable(kind model.Kind) (string, bool) {
	spec, err := s.model.kind(kind)
	if err != nil {
		return "", false
	}
	if spec.versioned() {
		return spec.identity.name, true
	}
	return spec.root.name, true
}

// VersionInfo is one stored version of a document: its number, the label a versioned definition declares, and when it
// was stored. A document updated in place keeps one version.
type VersionInfo struct {
	No      int64
	Label   string
	Created time.Time
}

// Versions lists a stored document's versions, oldest first.
func (s *DocumentStore) Versions(ctx context.Context, partnerID int64, key corpus.DocumentKey) ([]VersionInfo, error) {
	r, o, err := s.stored(ctx, partnerID, key)
	if err != nil {
		return nil, err
	}
	rows, err := r.rowsBy(ctx, "enterprise_object_version", "enterprise_object_id", []int64{o.id})
	if err != nil {
		return nil, fmt.Errorf("charter versions %s: %w", key, err)
	}
	out := make([]VersionInfo, len(rows))
	for i, row := range rows {
		out[i].No, _ = row["version_no"].(int64)
		out[i].Label, _ = row["definition_version"].(string)
		out[i].Created, _ = row["created_at"].(time.Time)
	}
	slices.SortFunc(out, func(a, b VersionInfo) int { return cmp.Compare(a.No, b.No) })
	return out, nil
}

// FetchVersion returns the version of a stored definition its label names; an unknown label is corpus.ErrNotFound.
func (s *DocumentStore) FetchVersion(ctx context.Context, partnerID int64, key corpus.DocumentKey, label string) (*model.Document, error) {
	r, o, err := s.stored(ctx, partnerID, key)
	if err != nil {
		return nil, err
	}
	versions, err := r.versions(ctx, []int64{o.id})
	if err != nil {
		return nil, fmt.Errorf("charter versions %s: %w", key, err)
	}
	for no, l := range versions[o.id] {
		if label != "" && l == label {
			o.version = no
		}
	}
	if o.version == 0 {
		return nil, fmt.Errorf("%w: %w: %s %q", corpus.ErrNotFound, ErrUnknownVersion, key, label)
	}
	docs, err := tenantStore{store: s, partnerID: partnerID}.documents(ctx, r, []storedObject{o})
	if err != nil {
		return nil, fmt.Errorf("charter document %s %q: %w", key, label, err)
	}
	return docs[0], nil
}

// stored finds a tenant's stored document.
func (s *DocumentStore) stored(ctx context.Context, partnerID int64, key corpus.DocumentKey) (*relationalReader, storedObject, error) {
	r, err := tenantStore{store: s, partnerID: partnerID}.reader()
	if err != nil {
		return nil, storedObject{}, err
	}
	o, found, err := r.document(ctx, key.Namespace, key.ID)
	if err != nil {
		return nil, storedObject{}, fmt.Errorf("charter document %s: %w", key, err)
	}
	if !found {
		return nil, storedObject{}, fmt.Errorf("%w: %s", corpus.ErrNotFound, key)
	}
	return r, o, nil
}

// DocumentQuery selects one page of a kind's current documents, ordered by namespace and identifier.
type DocumentQuery struct {
	Kind model.Kind
	// At is the day the current version's validity must contain; a document without validity is always effective.
	At time.Time
	// Parent keeps the documents directly under it: a unit's parent unit, a position's unit, a process's parent
	// process, or a task's process.
	Parent *model.ObjectRef
	// Prefix keeps the documents whose identifier or name starts with it, ignoring case.
	Prefix string
	// After is the last document of the previous page.
	After corpus.DocumentKey
	Limit int
}

// Page answers a document query in the store's own queries, so only the page is read.
func (s *DocumentStore) Page(ctx context.Context, partnerID int64, q DocumentQuery) ([]*model.Document, error) {
	t := tenantStore{store: s, partnerID: partnerID}
	r, err := t.reader()
	if err != nil {
		return nil, err
	}
	if q.Limit <= 0 || q.At.IsZero() {
		return nil, fmt.Errorf("%w: a positive limit and an effective time are required", ErrInvalidQuery)
	}
	spec, err := s.model.kind(q.Kind)
	if err != nil {
		return nil, err
	}
	day := time.Date(q.At.Year(), q.At.Month(), q.At.Day(), 0, 0, 0, 0, time.UTC)
	pattern := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(strings.ToLower(q.Prefix)) + "%"
	name, args := qPage, []any{spec.code, day, day, pattern, pattern, q.After.Namespace, q.After.ID}
	if q.Parent != nil {
		if spec.parent == "" {
			return nil, fmt.Errorf("%w: %s has no parent", ErrInvalidQuery, q.Kind)
		}
		ids, err := s.ObjectIDs(ctx, partnerID, *q.Parent)
		if err != nil {
			return nil, err
		}
		name, args = qPageUnder(spec.code), append(args, ids[0])
	}
	objs, err := r.objectRows(ctx, name, append(args, q.Limit)...)
	if err != nil {
		return nil, fmt.Errorf("charter documents %s: %w", q.Kind, err)
	}
	docs, err := t.documents(ctx, r, objs)
	if err != nil {
		return nil, fmt.Errorf("charter documents %s: %w", q.Kind, err)
	}
	return docs, nil
}

// Measurements returns, in window order, the tenant's evidence records measuring any of the outcome measures over a
// window within a closed period (CHR-PROC-011), selected by one query over the measurements.
func (s *DocumentStore) Measurements(ctx context.Context, partnerID int64, period model.Validity, measures ...corpus.DocumentKey) ([]model.EvidenceRecord, error) {
	t := tenantStore{store: s, partnerID: partnerID}
	r, err := t.reader()
	if err != nil || len(measures) == 0 {
		return nil, err
	}
	from, fromErr := time.Parse(dateLayout, string(period.From))
	to, toErr := time.Parse(dateLayout, string(period.To))
	if fromErr != nil || toErr != nil || to.Before(from) {
		return nil, fmt.Errorf("%w: a closed period is required", ErrInvalidQuery)
	}
	if _, err := s.model.kind(model.KindEvidenceRecord); err != nil {
		return nil, err
	}
	refs := make([]model.ObjectRef, len(measures))
	for i, m := range measures {
		refs[i] = model.ObjectRef{Kind: model.KindOutcomeMeasure, Namespace: m.Namespace, ID: m.ID}
	}
	ids, err := s.ObjectIDs(ctx, partnerID, refs...)
	if err != nil {
		return nil, err
	}
	rows, err := r.query(ctx, qMeasurements, partnerID, ids, from, to)
	if err != nil {
		return nil, fmt.Errorf("charter measurements: %w", err)
	}
	recordIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		id, ok := common.AsInt64OK(row[0])
		if !ok {
			return nil, fmt.Errorf("%w: measurement record %T", ErrDocumentIncomplete, row[0])
		}
		recordIDs = append(recordIDs, id)
	}
	byID, err := r.objectsByID(ctx, recordIDs)
	if err != nil {
		return nil, fmt.Errorf("charter measurements: %w", err)
	}
	docs, err := t.documents(ctx, r, slices.Collect(maps.Values(byID)))
	if err != nil {
		return nil, fmt.Errorf("charter measurements: %w", err)
	}
	out := make([]model.EvidenceRecord, len(docs))
	for i, d := range docs {
		if out[i], err = corpus.Decode[model.EvidenceRecord](d); err != nil {
			return nil, err
		}
	}
	slices.SortFunc(out, func(a, b model.EvidenceRecord) int {
		return cmp.Or(strings.Compare(string(a.Measurement.Window.From), string(b.Measurement.Window.From)),
			strings.Compare(a.Namespace, b.Namespace), strings.Compare(a.ID, b.ID))
	})
	return out, nil
}

// Evidence returns one tenant's append-only evidence store.
func (s *DocumentStore) Evidence(partnerID int64) evidence.Store {
	return tenantStore{store: s, partnerID: partnerID}
}

// Save validates and commits docs atomically, so documents that reference one another can be written together.
// Saving content equal to the stored document, or to a stored definition version, changes nothing.
func (s *DocumentStore) Save(ctx context.Context, partnerID int64, docs ...*model.Document) error {
	if partnerID <= 0 {
		return ErrNoTenant
	}
	normalized := make([]*model.Document, 0, len(docs))
	for _, d := range docs {
		n, err := canonical(d)
		if err != nil {
			return err
		}
		if slices.Contains(evidenceKinds, n.Kind) {
			return fmt.Errorf("%w: %s %s", ErrEvidenceDocument, n.Kind, n.ID)
		}
		normalized = append(normalized, n)
	}
	if len(normalized) == 0 {
		return nil
	}
	rows, err := s.decompose(normalized)
	if err != nil {
		return err
	}
	findings, err := s.additions.Validate(ctx, s.Source(partnerID), normalized...)
	if err != nil {
		return fmt.Errorf("charter documents: %w", err)
	}
	if len(findings) > 0 {
		return &InvalidDocumentsError{Findings: findings}
	}
	return s.write(ctx, partnerID, rows, false)
}

// decompose maps documents to rows, refusing a kind or a property the installed modules do not hold.
func (s *DocumentStore) decompose(docs []*model.Document) ([]*docRows, error) {
	rows := make([]*docRows, 0, len(docs))
	for _, d := range docs {
		if _, err := s.model.kind(d.Kind); err != nil {
			return nil, err
		}
		r, err := decompose(d)
		if err != nil {
			return nil, err
		}
		if err := s.model.admits(r); err != nil {
			return nil, err
		}
		rows = append(rows, r)
	}
	return rows, nil
}

func (s *DocumentStore) write(ctx context.Context, partnerID int64, rows []*docRows, appendOnly bool) (err error) {
	tx, err := s.db.BeginTx(ctx, s.model.queries)
	if err != nil {
		return fmt.Errorf("charter documents: begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			err = errors.Join(err, tx.Rollback(context.WithoutCancel(ctx)))
		}
	}()
	if err := newRelationalWriter(s.model, tx, partnerID).write(ctx, rows, appendOnly); err != nil {
		return fmt.Errorf("charter documents: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("charter documents: commit: %w", err)
	}
	committed = true
	return nil
}

// tenantStore is one partner's view: every kind for reading, evidence kinds for appending.
type tenantStore struct {
	store     *DocumentStore
	partnerID int64
}

var _ evidence.Store = tenantStore{}

func (t tenantStore) reader() (*relationalReader, error) {
	if t.store == nil {
		return nil, evidence.ErrNoStore
	}
	if t.partnerID <= 0 {
		return nil, ErrNoTenant
	}
	return &relationalReader{m: t.store.model, q: t.store.queries, partner: t.partnerID}, nil
}

// Append stores new evidence; a document already stored under its namespace and id is refused (evidence.ErrDuplicate).
func (t tenantStore) Append(ctx context.Context, d *model.Document) error {
	if _, err := t.reader(); err != nil {
		return err
	}
	if d == nil {
		return errors.New("cannot append a nil evidence document")
	}
	n, err := canonical(d)
	if err != nil {
		return err
	}
	if !slices.Contains(evidenceKinds, n.Kind) {
		return fmt.Errorf("%w: %s is not evidence", ErrInvalidDocument, n.Kind)
	}
	rows, err := t.store.decompose([]*model.Document{n})
	if err != nil {
		return err
	}
	return t.store.write(ctx, t.partnerID, rows, true)
}

func (t tenantStore) Fetch(ctx context.Context, key corpus.DocumentKey) (*model.Document, error) {
	r, err := t.reader()
	if err != nil {
		return nil, err
	}
	o, found, err := r.document(ctx, key.Namespace, key.ID)
	if err != nil {
		return nil, fmt.Errorf("charter document %s: %w", key, err)
	}
	if !found {
		return nil, fmt.Errorf("%w: %s", corpus.ErrNotFound, key)
	}
	docs, err := t.documents(ctx, r, []storedObject{o})
	if err != nil {
		return nil, fmt.Errorf("charter document %s: %w", key, err)
	}
	return docs[0], nil
}

func (t tenantStore) List(ctx context.Context, kind model.Kind) ([]*model.Document, error) {
	r, err := t.reader()
	if err != nil {
		return nil, err
	}
	spec, err := t.store.model.kind(kind)
	if err != nil {
		return nil, err
	}
	objs, err := r.objectRows(ctx, qObjectsOfKind, spec.code)
	if err != nil {
		return nil, fmt.Errorf("charter documents %s: %w", kind, err)
	}
	sort.Slice(objs, func(i, j int) bool {
		if objs[i].key.NS != objs[j].key.NS {
			return objs[i].key.NS < objs[j].key.NS
		}
		return objs[i].key.ID < objs[j].key.ID
	})
	docs, err := t.documents(ctx, r, objs)
	if err != nil {
		return nil, fmt.Errorf("charter documents %s: %w", kind, err)
	}
	return docs, nil
}

// documents rebuilds stored objects of one kind at their current versions.
func (t tenantStore) documents(ctx context.Context, r *relationalReader, objs []storedObject) ([]*model.Document, error) {
	if len(objs) == 0 {
		return nil, nil
	}
	spec, err := t.store.model.kind(model.Kind(kindCodes.words[objs[0].code]))
	if err != nil {
		return nil, err
	}
	if err := r.current(ctx, objs); err != nil {
		return nil, err
	}
	loaded, err := r.load(ctx, spec, objs)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Document, 0, len(loaded))
	for _, l := range loaded {
		c := &composer{ns: l.object.key.NS, rows: l.rows, kindOf: l.kindOf, exponent: l.exponent, agent: l.agent}
		value, err := c.compose(spec, l.object.key.ID, l.enterprise)
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		d, err := (corpus.Parser{}).Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDocumentIncomplete, err)
		}
		out = append(out, d)
	}
	return out, nil
}

// canonical re-parses a document so its value decodes numbers exactly.
func canonical(d *model.Document) (*model.Document, error) {
	if d == nil {
		return nil, errors.New("cannot save a nil document")
	}
	value := d.Value
	if value == nil {
		parsed, err := (corpus.Parser{}).Parse(d.Raw)
		if err != nil {
			return nil, err
		}
		value = parsed.Value
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return (corpus.Parser{}).Parse(raw)
}
