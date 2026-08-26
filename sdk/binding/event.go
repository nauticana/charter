package binding

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nauticana/charter/sdk/model"
)

type DeliveryStatus string

const (
	DeliveryAccepted  DeliveryStatus = "accepted"
	DeliveryDuplicate DeliveryStatus = "duplicate"
	DeliveryRejected  DeliveryStatus = "rejected"
)

const (
	DeliverAtMostOnce  = "at-most-once"
	DeliverAtLeastOnce = "at-least-once"
	DeliverExactlyOnce = "exactly-once"
)

// Delivery is one external event as it reaches the Charter trigger an event binding maps it to.
type Delivery struct {
	Binding    model.Ref
	EventID    string
	Trigger    string
	Payload    any
	ReceivedAt time.Time
}

// Receipt reports how a delivery was treated; a rejected delivery names why (CHR-SEC-008).
type Receipt struct {
	Status DeliveryStatus
	Reason string
}

// DeliveryLedger claims event ids atomically so concurrent deliveries cannot both run. A successful delivery is
// completed; an at-least-once delivery that fails is released so it may be retried.
type DeliveryLedger interface {
	Begin(ctx context.Context, binding model.Ref, eventID string) (duplicate bool, err error)
	Complete(ctx context.Context, binding model.Ref, eventID string) error
	Release(ctx context.Context, binding model.Ref, eventID string) error
}

// Handler is the runtime's reaction to a delivered trigger. An at-least-once handler error releases the claim for
// redelivery; an at-most-once claim remains consumed even when handling fails.
type Handler interface {
	Handle(ctx context.Context, d Delivery) error
}

// AbstractEventConsumer applies an event binding's declared delivery semantics before an abstract Handler runs.
// At-most-once and at-least-once deliveries are claimed atomically; exactly-once is rejected because it requires the
// handler and ledger to commit in one transaction, which this abstraction cannot guarantee.
type AbstractEventConsumer struct {
	Bindings Provider
	Ledger   DeliveryLedger
	Handler  Handler
}

func (c *AbstractEventConsumer) Deliver(ctx context.Context, ownerNamespace string, bindingRef model.Ref, eventID string, payload any, at time.Time) (Receipt, error) {
	if c.Bindings == nil || c.Handler == nil {
		return Receipt{Status: DeliveryRejected, Reason: "event consumer is not fully composed"}, nil
	}
	b, err := c.Bindings.EventBinding(ctx, ownerNamespace, bindingRef)
	if err != nil {
		return Receipt{Status: DeliveryRejected, Reason: err.Error()}, nil
	}
	if b.LifecycleState != "" && b.LifecycleState != model.LifecycleActive {
		return Receipt{Status: DeliveryRejected, Reason: fmt.Sprintf("event binding %s is %s", b.ID, b.LifecycleState)}, nil
	}
	key := model.Ref{Namespace: b.Namespace, ID: b.ID}
	switch b.DeliverySemantics {
	case DeliverAtMostOnce, DeliverAtLeastOnce:
		if eventID == "" {
			return Receipt{Status: DeliveryRejected, Reason: b.DeliverySemantics + " delivery without an event id cannot be deduplicated"}, nil
		}
		if c.Ledger == nil {
			return Receipt{Status: DeliveryRejected, Reason: b.DeliverySemantics + " delivery requires a delivery ledger"}, nil
		}
		duplicate, err := c.Ledger.Begin(ctx, key, eventID)
		if err != nil {
			return Receipt{Status: DeliveryRejected, Reason: err.Error()}, err
		}
		if duplicate {
			return Receipt{Status: DeliveryDuplicate, Reason: "event " + eventID + " was already claimed"}, nil
		}
	case DeliverExactlyOnce:
		return Receipt{Status: DeliveryRejected, Reason: "exactly-once delivery requires an atomic handler and ledger"}, nil
	default:
		return Receipt{Status: DeliveryRejected, Reason: fmt.Sprintf("event binding %s declares unsupported delivery semantics %q", b.ID, b.DeliverySemantics)}, nil
	}
	d := Delivery{Binding: key, EventID: eventID, Trigger: b.CharterTrigger, Payload: payload, ReceivedAt: at}
	if err := c.Handler.Handle(ctx, d); err != nil {
		if b.DeliverySemantics == DeliverAtLeastOnce {
			if releaseErr := c.Ledger.Release(ctx, key, eventID); releaseErr != nil {
				return Receipt{Status: DeliveryRejected, Reason: err.Error() + "; delivery claim release failed: " + releaseErr.Error()}, errors.Join(err, releaseErr)
			}
		}
		return Receipt{Status: DeliveryRejected, Reason: err.Error()}, err
	}
	if b.DeliverySemantics == DeliverAtMostOnce || b.DeliverySemantics == DeliverAtLeastOnce {
		if err := c.Ledger.Complete(ctx, key, eventID); err != nil {
			return Receipt{Status: DeliveryAccepted, Reason: "handled, but the delivery ledger failed: " + err.Error()}, err
		}
	}
	return Receipt{Status: DeliveryAccepted}, nil
}

// BaseMemoryDeliveryLedger keeps handled event ids in memory.
type BaseMemoryDeliveryLedger struct {
	mu   sync.Mutex
	seen map[string]bool
}

var _ DeliveryLedger = (*BaseMemoryDeliveryLedger)(nil)

func NewBaseMemoryDeliveryLedger() *BaseMemoryDeliveryLedger {
	return &BaseMemoryDeliveryLedger{seen: map[string]bool{}}
}

func (l *BaseMemoryDeliveryLedger) Begin(_ context.Context, binding model.Ref, eventID string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := binding.Namespace + ":" + binding.ID + "|" + eventID
	if l.seen[key] {
		return true, nil
	}
	l.seen[key] = true
	return false, nil
}

func (l *BaseMemoryDeliveryLedger) Complete(_ context.Context, binding model.Ref, eventID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if eventID == "" {
		return errors.New("event id is required")
	}
	if !l.seen[binding.Namespace+":"+binding.ID+"|"+eventID] {
		return errors.New("event id was not begun")
	}
	return nil
}

func (l *BaseMemoryDeliveryLedger) Release(_ context.Context, binding model.Ref, eventID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.seen, binding.Namespace+":"+binding.ID+"|"+eventID)
	return nil
}
