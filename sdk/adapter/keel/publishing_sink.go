package keel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nauticana/keel/common"
	"github.com/nauticana/keel/port"

	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/model"
)

var ErrPublish = errors.New("evidence appended but not published")

// PublishingSink appends through the next sink, then publishes the document, redacted, as an event with kind, namespace,
// id, and request-id attributes. A publish failure surfaces as ErrPublish after the append already succeeded (CHR-SEC-007).
type PublishingSink struct {
	Next      evidence.Sink
	Publisher port.MessagePublisher
	Topic     string
	Redactor  evidence.Redactor
}

var _ evidence.Sink = (*PublishingSink)(nil)

func (s *PublishingSink) Action(ctx context.Context, a model.ActionRecord) error {
	if s.Next == nil {
		return evidence.ErrNoStore
	}
	a.Kind = model.KindActionRecord
	return s.publish(ctx, s.Next.Action(ctx, a), a.Envelope, a)
}

func (s *PublishingSink) Record(ctx context.Context, r model.EvidenceRecord) error {
	if s.Next == nil {
		return evidence.ErrNoStore
	}
	r.Kind = model.KindEvidenceRecord
	return s.publish(ctx, s.Next.Record(ctx, r), r.Envelope, r)
}

func (s *PublishingSink) Exception(ctx context.Context, e model.ExceptionRecord) error {
	if s.Next == nil {
		return evidence.ErrNoStore
	}
	e.Kind = model.KindExceptionRecord
	return s.publish(ctx, s.Next.Exception(ctx, e), e.Envelope, e)
}

func (s *PublishingSink) Escalation(ctx context.Context, e model.Escalation) error {
	if s.Next == nil {
		return evidence.ErrNoStore
	}
	e.Kind = model.KindEscalation
	return s.publish(ctx, s.Next.Escalation(ctx, e), e.Envelope, e)
}

func (s *PublishingSink) Bundle(ctx context.Context, b model.EvidenceBundle) error {
	if s.Next == nil {
		return evidence.ErrNoStore
	}
	b.Kind = model.KindEvidenceBundle
	return s.publish(ctx, s.Next.Bundle(ctx, b), b.Envelope, b)
}

func (s *PublishingSink) publish(ctx context.Context, appended error, env model.Envelope, doc any) error {
	if appended != nil {
		return appended
	}
	if s.Publisher == nil {
		return fmt.Errorf("%w: no publisher", ErrPublish)
	}
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPublish, err)
	}
	if s.Redactor != nil {
		if data, err = s.Redactor.Redact(data); err != nil {
			return fmt.Errorf("%w: redaction: %v", ErrPublish, err)
		}
	}
	attributes := map[string]string{"kind": string(env.Kind), "namespace": env.Namespace, "id": env.ID}
	if requestID := common.RequestIDFromContext(ctx); requestID != "" {
		attributes["request_id"] = requestID
	}
	if err := s.Publisher.Publish(ctx, s.Topic, data, attributes); err != nil {
		return fmt.Errorf("%w: %v", ErrPublish, err)
	}
	return nil
}
