package keel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/nauticana/keel/common"
	kmodel "github.com/nauticana/keel/model"
	"github.com/nauticana/keel/port"

	"github.com/nauticana/charter/sdk/corpus"
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
	ErrStoredDocument    = errors.New("stored document is inconsistent")
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

const (
	qDocumentClaim     = "charter_document_claim"
	qRevisions         = "charter_document_revisions"
	qRevisionInsert    = "charter_document_revision_insert"
	qDocumentCurrent   = "charter_document_current"
	qDocumentsOfKind   = "charter_documents_of_kind"
	currentRevisionSQL = `r.revision = (SELECT MAX(x.revision) FROM charter_document_revision x
       WHERE x.partner_id = r.partner_id AND x.namespace = r.namespace AND x.document_id = r.document_id)`
)

var documentQueries = map[string]string{
	qDocumentClaim: `
INSERT INTO charter_document (partner_id, namespace, document_id, document_kind) VALUES (?, ?, ?, ?)
ON CONFLICT (partner_id, namespace, document_id) DO UPDATE
 SET document_kind = charter_document.document_kind
RETURNING document_kind`,
	qRevisions: `
SELECT revision, definition_version, content_digest, content FROM charter_document_revision
 WHERE partner_id = ? AND namespace = ? AND document_id = ?
 ORDER BY revision DESC`,
	qRevisionInsert: `
INSERT INTO charter_document_revision (partner_id, namespace, document_id, revision, definition_version, content, content_digest)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
	qDocumentCurrent: `
SELECT r.content, r.content_digest FROM charter_document_revision r
 WHERE r.partner_id = ? AND r.namespace = ? AND r.document_id = ? AND ` + currentRevisionSQL,
	qDocumentsOfKind: `
SELECT r.content, r.content_digest FROM charter_document d
  JOIN charter_document_revision r ON r.partner_id = d.partner_id AND r.namespace = d.namespace AND r.document_id = d.document_id
 WHERE d.partner_id = ? AND d.document_kind = ? AND ` + currentRevisionSQL + `
 ORDER BY d.namespace, d.document_id`,
}

// DocumentStore keeps a tenant's Charter documents in the charter_document schema module. Every save appends a
// revision; the latest is current. A document keeps its kind and enterprise so an update cannot invalidate inbound
// references. A versioned definition never changes once stored: different content under a
// stored version is refused, so a changed definition must declare a new version (CHR-PROC-005, CHR-AGENT-006).
// Documents are validated for structure and references before commit, fetching only the documents they reference.
type DocumentStore struct {
	db        port.DatabaseRepository
	queries   port.QueryService
	additions *validate.Additions
}

func NewDocumentStore(ctx context.Context, db port.DatabaseRepository) (*DocumentStore, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	additions, err := validate.NewAdditions()
	if err != nil {
		return nil, err
	}
	queries := db.GetQueryService(ctx, documentQueries)
	if queries == nil {
		return nil, fmt.Errorf("%w: no query service", ErrNoDatabase)
	}
	return &DocumentStore{db: db, queries: queries, additions: additions}, nil
}

// Source returns the current documents of one tenant for Charter providers.
func (s *DocumentStore) Source(partnerID int64) corpus.Source {
	return tenantDocuments{store: s, partnerID: partnerID}
}

// Save validates and commits docs atomically, so documents that reference one another can be written together.
// Saving content identical to the current revision changes nothing.
func (s *DocumentStore) Save(ctx context.Context, partnerID int64, docs ...*model.Document) (err error) {
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
	findings, err := s.additions.Validate(ctx, s.Source(partnerID), normalized...)
	if err != nil {
		return fmt.Errorf("charter documents: %w", err)
	}
	if len(findings) > 0 {
		return &InvalidDocumentsError{Findings: findings}
	}
	tx, err := s.db.BeginTx(ctx, documentQueries)
	if err != nil {
		return fmt.Errorf("charter documents: begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			err = errors.Join(err, tx.Rollback(context.WithoutCancel(ctx)))
		}
	}()
	for _, d := range normalized {
		if err := s.append(ctx, tx, partnerID, d); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("charter documents: commit: %w", err)
	}
	committed = true
	return nil
}

func (s *DocumentStore) append(ctx context.Context, tx port.TxQueryService, partnerID int64, d *model.Document) error {
	key := corpus.DocumentKey{Namespace: d.Namespace, ID: d.ID}
	kinds, err := tx.Query(ctx, qDocumentClaim, partnerID, d.Namespace, d.ID, string(d.Kind))
	if err != nil {
		return fmt.Errorf("charter document %s: %w", key, err)
	}
	if len(kinds.Rows) != 1 {
		return fmt.Errorf("charter document %s: claim returned %d rows", key, len(kinds.Rows))
	}
	if stored := common.AsString(kinds.Rows[0][0]); stored != string(d.Kind) {
		return fmt.Errorf("%w: %s is %s, not %s", ErrKindChanged, key, stored, d.Kind)
	}
	revisions, err := tx.Query(ctx, qRevisions, partnerID, d.Namespace, d.ID)
	if err != nil {
		return fmt.Errorf("charter document %s revisions: %w", key, err)
	}
	digest := digestOf(d.Raw)
	version, versioned := corpus.DefinitionVersion(d)
	next := int64(1)
	for i, row := range revisions.Rows {
		if i == 0 {
			if common.AsString(row[2]) == digest {
				return nil
			}
			current, err := documents(&kmodel.QueryResult{Rows: [][]any{{row[3], row[2]}}})
			if err != nil || len(current) != 1 {
				return fmt.Errorf("charter document %s current revision: %w", key, errors.Join(ErrStoredDocument, err))
			}
			if !sameEnterprise(current[0], d) {
				return fmt.Errorf("%w: %s", ErrEnterpriseChanged, key)
			}
			next = common.AsInt64(row[0]) + 1
		}
		if versioned && common.AsString(row[1]) == version {
			return fmt.Errorf("%w: %s %s version %q is stored with other content", ErrPublishedVersion, d.Kind, key, version)
		}
	}
	var definitionVersion any
	if versioned {
		definitionVersion = version
	}
	if _, err := tx.Query(ctx, qRevisionInsert, partnerID, d.Namespace, d.ID, next, definitionVersion, string(d.Raw), digest); err != nil {
		return fmt.Errorf("charter document %s revision %d: %w", key, next, err)
	}
	return nil
}

type tenantDocuments struct {
	store     *DocumentStore
	partnerID int64
}

var _ corpus.Source = tenantDocuments{}

func (t tenantDocuments) Fetch(ctx context.Context, key corpus.DocumentKey) (*model.Document, error) {
	if t.partnerID <= 0 {
		return nil, ErrNoTenant
	}
	result, err := t.store.queries.Query(ctx, qDocumentCurrent, t.partnerID, key.Namespace, key.ID)
	if err != nil {
		return nil, fmt.Errorf("charter document %s: %w", key, err)
	}
	docs, err := documents(result)
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("%w: %s", corpus.ErrNotFound, key)
	}
	if (corpus.DocumentKey{Namespace: docs[0].Namespace, ID: docs[0].ID}) != key {
		return nil, fmt.Errorf("%w: %s returned %s:%s", ErrStoredDocument, key, docs[0].Namespace, docs[0].ID)
	}
	return docs[0], nil
}

func (t tenantDocuments) List(ctx context.Context, kind model.Kind) ([]*model.Document, error) {
	if t.partnerID <= 0 {
		return nil, ErrNoTenant
	}
	result, err := t.store.queries.Query(ctx, qDocumentsOfKind, t.partnerID, string(kind))
	if err != nil {
		return nil, fmt.Errorf("charter documents %s: %w", kind, err)
	}
	docs, err := documents(result)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		if d.Kind != kind {
			return nil, fmt.Errorf("%w: %s:%s is %s, listed as %s", ErrStoredDocument, d.Namespace, d.ID, d.Kind, kind)
		}
	}
	return docs, nil
}

// documents parses content rows and verifies each against its stored digest.
func documents(result *kmodel.QueryResult) ([]*model.Document, error) {
	if result == nil {
		return nil, nil
	}
	out := make([]*model.Document, 0, len(result.Rows))
	for _, row := range result.Rows {
		content := []byte(common.AsString(row[0]))
		if digestOf(content) != common.AsString(row[1]) {
			return nil, fmt.Errorf("%w: content does not match its digest", ErrStoredDocument)
		}
		d, err := (corpus.Parser{}).Parse(content)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrStoredDocument, err)
		}
		out = append(out, d)
	}
	return out, nil
}

// canonical re-encodes a document with sorted keys so equal content always has the same digest.
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

func digestOf(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func sameEnterprise(a, b *model.Document) bool {
	if a.EnterpriseID == nil || b.EnterpriseID == nil {
		return a.EnterpriseID == nil && b.EnterpriseID == nil
	}
	return corpus.KeyOf(a.Namespace, *a.EnterpriseID) == corpus.KeyOf(b.Namespace, *b.EnterpriseID)
}
