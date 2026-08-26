package capability

import (
	"context"
	"sync"
)

type LedgerState string

const (
	LedgerNew       LedgerState = ""
	LedgerInFlight  LedgerState = "in-flight"
	LedgerUnknown   LedgerState = "unknown"
	LedgerCompleted LedgerState = "completed"
)

type LedgerEntry struct {
	State  LedgerState
	Result *Result
}

// Ledger records mutating invocations by idempotency key so replays return the prior result and ambiguous outcomes
// block retries until reconciled (CHR-CAP-004, CHR-SEC-008).
type Ledger interface {
	// Begin marks a new key in-flight and returns LedgerNew; otherwise it returns the existing entry untouched.
	Begin(ctx context.Context, key string) (LedgerEntry, error)
	Complete(ctx context.Context, key string, r Result) error
	// Release forgets a key whose operation provably did not execute.
	Release(ctx context.Context, key string) error
	MarkUnknown(ctx context.Context, key string) error
}

// BaseMemoryLedger keeps ledger entries in memory.
type BaseMemoryLedger struct {
	mu      sync.Mutex
	entries map[string]LedgerEntry
}

var _ Ledger = (*BaseMemoryLedger)(nil)

func NewBaseMemoryLedger() *BaseMemoryLedger {
	return &BaseMemoryLedger{entries: map[string]LedgerEntry{}}
}

func (l *BaseMemoryLedger) Begin(_ context.Context, key string) (LedgerEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e, ok := l.entries[key]; ok {
		return e, nil
	}
	l.entries[key] = LedgerEntry{State: LedgerInFlight}
	return LedgerEntry{State: LedgerNew}, nil
}

func (l *BaseMemoryLedger) Complete(_ context.Context, key string, r Result) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries[key] = LedgerEntry{State: LedgerCompleted, Result: &r}
	return nil
}

func (l *BaseMemoryLedger) Release(_ context.Context, key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
	return nil
}

func (l *BaseMemoryLedger) MarkUnknown(_ context.Context, key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries[key] = LedgerEntry{State: LedgerUnknown}
	return nil
}
