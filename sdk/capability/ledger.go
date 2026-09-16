package capability

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"sync"
)

type LedgerState string

const (
	LedgerNew       LedgerState = ""
	LedgerInFlight  LedgerState = "in-flight"
	LedgerUnknown   LedgerState = "unknown"
	LedgerCompleted LedgerState = "completed"
)

var (
	ErrLedgerFence      = errors.New("ledger: fence does not hold the key")
	ErrLedgerTransition = errors.New("ledger: invalid state transition")
)

// LedgerEntry is the recorded state of one key. Fence is set only when Begin granted the claim to the caller; every
// later write must present it.
type LedgerEntry struct {
	State  LedgerState
	Result *Result
	Fence  string
}

// Ledger records mutating invocations by idempotency key so replays return the prior result and ambiguous outcomes
// block retries until reconciled (CHR-CAP-004, CHR-SEC-008).
type Ledger interface {
	// Begin claims a new key and returns LedgerNew with a Fence; otherwise it returns the existing entry without one.
	Begin(ctx context.Context, key string) (LedgerEntry, error)
	Complete(ctx context.Context, key, fence string, r Result) error
	// Release forgets a key whose operation provably did not execute.
	Release(ctx context.Context, key, fence string) error
	MarkUnknown(ctx context.Context, key, fence string) error
}

// BaseMemoryLedger keeps ledger entries in memory.
type BaseMemoryLedger struct {
	mu      sync.Mutex
	seq     uint64
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
		var result *Result
		if e.Result != nil {
			stored := *e.Result
			stored.LedgerFence = ""
			result = &stored
		}
		return LedgerEntry{State: e.State, Result: result}, nil
	}
	if l.entries == nil {
		l.entries = map[string]LedgerEntry{}
	}
	l.seq++
	fence := strconv.FormatUint(l.seq, 10)
	l.entries[key] = LedgerEntry{State: LedgerInFlight, Fence: fence}
	return LedgerEntry{State: LedgerNew, Fence: fence}, nil
}

func (l *BaseMemoryLedger) Complete(_ context.Context, key, fence string, r Result) error {
	r.LedgerFence = ""
	r.Err = nil
	return l.write(key, fence, LedgerEntry{State: LedgerCompleted, Result: &r, Fence: fence})
}

func (l *BaseMemoryLedger) MarkUnknown(_ context.Context, key, fence string) error {
	return l.write(key, fence, LedgerEntry{State: LedgerUnknown, Fence: fence})
}

func (l *BaseMemoryLedger) Release(_ context.Context, key, fence string) error {
	return l.write(key, fence, LedgerEntry{})
}

// write replaces the entry of a held key, or forgets the key when next is LedgerNew.
func (l *BaseMemoryLedger) write(key, fence string, next LedgerEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[key]
	if !ok || fence == "" || e.Fence != fence {
		return ErrLedgerFence
	}
	if e.State == LedgerCompleted {
		if next.State == LedgerCompleted && reflect.DeepEqual(e.Result, next.Result) {
			return nil
		}
		return ErrLedgerTransition
	}
	if e.State != LedgerInFlight && e.State != LedgerUnknown {
		return ErrLedgerTransition
	}
	if next.State == LedgerNew {
		delete(l.entries, key)
	} else {
		l.entries[key] = next
	}
	return nil
}
