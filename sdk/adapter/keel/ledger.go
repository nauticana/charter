package keel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	kmodel "github.com/nauticana/keel/model"
	"github.com/nauticana/keel/port"

	"github.com/nauticana/charter/sdk/capability"
)

var ErrNoLedger = errors.New("no keel idempotency ledger")

// Ledger backs capability.Ledger with keel's port.IdempotencyLedger, storing each completed Result as JSON.
// Compose it with a zero lease: the invoker runs its transport synchronously and never renews a claim.
type Ledger struct {
	Keel port.IdempotencyLedger
}

var _ capability.Ledger = (*Ledger)(nil)

func (l *Ledger) Begin(ctx context.Context, key string) (capability.LedgerEntry, error) {
	if l.Keel == nil {
		return capability.LedgerEntry{}, ErrNoLedger
	}
	e, err := l.Keel.Begin(ctx, key)
	if err != nil {
		return capability.LedgerEntry{}, err
	}
	entry := capability.LedgerEntry{State: capability.LedgerState(e.State), Fence: e.Fence}
	if e.State == kmodel.LedgerCompleted {
		var r capability.Result
		if err := json.Unmarshal(e.Result, &r); err != nil {
			return capability.LedgerEntry{}, fmt.Errorf("stored result of %s: %w", key, err)
		}
		entry.Result = &r
	}
	return entry, nil
}

func (l *Ledger) Complete(ctx context.Context, key, fence string, r capability.Result) error {
	if l.Keel == nil {
		return ErrNoLedger
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return l.Keel.Complete(ctx, key, fence, data)
}

func (l *Ledger) Release(ctx context.Context, key, fence string) error {
	if l.Keel == nil {
		return ErrNoLedger
	}
	return l.Keel.Release(ctx, key, fence)
}

func (l *Ledger) MarkUnknown(ctx context.Context, key, fence string) error {
	if l.Keel == nil {
		return ErrNoLedger
	}
	return l.Keel.MarkUnknown(ctx, key, fence)
}
