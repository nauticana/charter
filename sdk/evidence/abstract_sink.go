package evidence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// AbstractSink implements append-only writes, supersession links, bundle integrity, and secret redaction over an
// abstract Store; a nil Redactor appends evidence verbatim.
type AbstractSink struct {
	Store    Store
	Digester Digester
	Redactor Redactor
}

var _ Sink = (*AbstractSink)(nil)

func (s *AbstractSink) Action(ctx context.Context, a model.ActionRecord) error {
	a.Kind = model.KindActionRecord
	return s.append(ctx, a.Envelope, a)
}

// Record appends a record; a correction must name an existing evidence record it supersedes, which stays addressable (CHR-EVID-004).
func (s *AbstractSink) Record(ctx context.Context, r model.EvidenceRecord) error {
	r.Kind = model.KindEvidenceRecord
	if r.Supersedes != nil {
		if err := s.ready(); err != nil {
			return err
		}
		d, err := s.Store.Fetch(ctx, corpus.KeyOf(r.Namespace, *r.Supersedes))
		if err != nil || d.Kind != model.KindEvidenceRecord {
			return fmt.Errorf("%w: %s", ErrSupersedes, r.Supersedes.ID)
		}
	}
	return s.append(ctx, r.Envelope, r)
}

func (s *AbstractSink) Exception(ctx context.Context, e model.ExceptionRecord) error {
	e.Kind = model.KindExceptionRecord
	return s.append(ctx, e.Envelope, e)
}

func (s *AbstractSink) Escalation(ctx context.Context, e model.Escalation) error {
	e.Kind = model.KindEscalation
	return s.append(ctx, e.Envelope, e)
}

// Bundle appends a bundle whose records are all stored; an absent integrity value is computed, a present one is verified (CHR-EVID-005).
func (s *AbstractSink) Bundle(ctx context.Context, b model.EvidenceBundle) error {
	if err := s.ready(); err != nil {
		return err
	}
	b.Kind = model.KindEvidenceBundle
	v := Verifier{Source: s.Store, Digester: s.Digester}
	if b.Integrity.Method == "" {
		b.Integrity.Method = s.Digester.Method()
	}
	if b.Integrity.Value == "" {
		if b.Integrity.Method != s.Digester.Method() {
			return fmt.Errorf("%w: cannot compute method %q", ErrIntegrity, b.Integrity.Method)
		}
		value, err := v.chain(ctx, b)
		if err != nil {
			return err
		}
		b.Integrity.Value = value
	} else if err := v.check(ctx, b); err != nil {
		return err
	}
	return s.append(ctx, b.Envelope, b)
}

// Assemble fills an empty bundle with every stored record about its subject, in order, and appends it with a computed
// integrity value (CHR-EVID-005).
func (s *AbstractSink) Assemble(ctx context.Context, b model.EvidenceBundle) (model.EvidenceBundle, error) {
	if err := s.ready(); err != nil {
		return b, err
	}
	if len(b.RecordIDs) == 0 {
		refs, err := (Queries{Provider: NewBaseProvider(s.Store)}).RecordsAbout(ctx, b.Namespace, b.Subject)
		if err != nil {
			return b, err
		}
		if len(refs) == 0 {
			return b, fmt.Errorf("%w: nothing recorded about %s %s", ErrBundleRecord, b.Subject.Kind, b.Subject.ID)
		}
		b.RecordIDs = refs
	}
	b.Kind = model.KindEvidenceBundle
	if b.Integrity.Method == "" {
		b.Integrity.Method = s.Digester.Method()
	}
	if b.Integrity.Value == "" {
		value, err := (Verifier{Source: s.Store, Digester: s.Digester}).chain(ctx, b)
		if err != nil {
			return b, err
		}
		b.Integrity.Value = value
	}
	return b, s.Bundle(ctx, b)
}

func (s *AbstractSink) ready() error {
	if s.Store == nil || s.Digester == nil {
		return ErrNoStore
	}
	return nil
}

func (s *AbstractSink) append(ctx context.Context, env model.Envelope, v any) error {
	if err := s.ready(); err != nil {
		return err
	}
	if env.Namespace == "" || env.ID == "" {
		return fmt.Errorf("evidence %s requires namespace and id", env.Kind)
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if s.Redactor != nil {
		if raw, err = s.Redactor.Redact(raw); err != nil {
			return fmt.Errorf("redaction: %w", err)
		}
	}
	d := &model.Document{Envelope: env, Raw: raw}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&d.Value); err != nil {
		return err
	}
	return s.Store.Append(ctx, d)
}
