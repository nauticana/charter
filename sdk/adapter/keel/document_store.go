package keel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

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
