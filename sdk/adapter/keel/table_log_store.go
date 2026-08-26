package keel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	kmodel "github.com/nauticana/keel/model"
	"github.com/nauticana/keel/port"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/model"
)

const appendAction = "I"

var evidenceKinds = []model.Kind{model.KindActionRecord, model.KindEvidenceRecord, model.KindEvidenceBundle, model.KindExceptionRecord, model.KindEscalation}

var farFuture = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

// TableLogStore keeps Charter evidence in keel's table change log: one appended change row per document, keyed by
// namespace and id, never updated or deleted. PartnerID and OwnerUserID are the scope already resolved by the caller;
// zero retains keel's unrestricted convention. The file logger cannot answer FindChanges, so appends and reads fail closed on it.
type TableLogStore struct {
	Logger      port.TableLogger
	Prefix      string
	PartnerID   int64
	OwnerUserID int
	CreatedBy   int
}

var _ evidence.Store = (*TableLogStore)(nil)

func (s *TableLogStore) table(kind model.Kind) string {
	prefix := s.Prefix
	if prefix == "" {
		prefix = "charter_"
	}
	return prefix + strings.ToLower(string(kind))
}

func (s *TableLogStore) Append(ctx context.Context, d *model.Document) error {
	if s.Logger == nil {
		return evidence.ErrNoStore
	}
	key := corpus.DocumentKey{Namespace: d.Namespace, ID: d.ID}
	if _, err := s.Fetch(ctx, key); err == nil {
		return fmt.Errorf("%w: %s", evidence.ErrDuplicate, key)
	} else if !errors.Is(err, corpus.ErrNotFound) {
		return err
	}
	value, ok := d.Value.(map[string]any)
	if !ok {
		dec := json.NewDecoder(bytes.NewReader(d.Raw))
		dec.UseNumber()
		if err := dec.Decode(&value); err != nil {
			return err
		}
	}
	return s.Logger.LogChange(ctx, &kmodel.TableChangeLog{TableName: s.table(d.Kind), RecordKey: key.String(), Action: appendAction,
		OldData: value, PartnerID: s.PartnerID, OwnerUserID: s.OwnerUserID, CreatedAt: time.Now().UTC(), CreatedBy: s.CreatedBy})
}

func (s *TableLogStore) Fetch(ctx context.Context, key corpus.DocumentKey) (*model.Document, error) {
	if s.Logger == nil {
		return nil, evidence.ErrNoStore
	}
	for _, kind := range evidenceKinds {
		filter := port.ChangeFilter{TableName: s.table(kind), RecordKey: key.String(), Action: appendAction, Endda: farFuture}
		changes, err := s.Logger.FindChanges(ctx, filter, s.PartnerID, s.OwnerUserID)
		if err != nil {
			return nil, fmt.Errorf("table log %s: %w", s.table(kind), err)
		}
		if len(changes) > 0 {
			return document(changes[0])
		}
	}
	return nil, fmt.Errorf("%w: %s", corpus.ErrNotFound, key)
}

func (s *TableLogStore) List(ctx context.Context, kind model.Kind) ([]*model.Document, error) {
	if s.Logger == nil {
		return nil, evidence.ErrNoStore
	}
	filter := port.ChangeFilter{TableName: s.table(kind), Action: appendAction, Endda: farFuture}
	changes, err := s.Logger.FindChanges(ctx, filter, s.PartnerID, s.OwnerUserID)
	if err != nil {
		return nil, fmt.Errorf("table log %s: %w", s.table(kind), err)
	}
	out := make([]*model.Document, 0, len(changes))
	for _, c := range changes {
		d, err := document(c)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

// document rebuilds a Charter document from a change row; sorted-key marshalling keeps digests stable across reads.
func document(c *kmodel.TableChangeLog) (*model.Document, error) {
	raw, err := json.Marshal(c.OldData)
	if err != nil {
		return nil, fmt.Errorf("change %d: %w", c.ID, err)
	}
	return (corpus.Parser{}).Parse(raw)
}
