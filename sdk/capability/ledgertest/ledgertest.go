// Package ledgertest is the contract suite every capability.Ledger implementation runs from its own tests.
package ledgertest

import (
	"context"
	"testing"

	"github.com/nauticana/charter/sdk/capability"
)

// Run checks claim, fencing, completion, release, unknown, and reclaim semantics. newLedger returns an empty ledger.
// Refusals are checked as errors, not specific sentinels, since adapters may return their store's errors.
func Run(t *testing.T, newLedger func(t *testing.T) capability.Ledger) {
	ctx := context.Background()
	done := capability.Result{Status: capability.StatusExecuted, Outcome: "done"}

	t.Run("a new key is claimed once with a fence", func(t *testing.T) {
		l := newLedger(t)
		first := begin(t, l, "k")
		if first.State != capability.LedgerNew || first.Fence == "" {
			t.Fatalf("claim: %+v", first)
		}
		if again := begin(t, l, "k"); again.State != capability.LedgerInFlight || again.Fence != "" || again.Result != nil {
			t.Fatalf("concurrent claim: %+v", again)
		}
	})

	t.Run("a stale fence is refused", func(t *testing.T) {
		l := newLedger(t)
		begin(t, l, "k")
		if err := l.Complete(ctx, "k", "stale", done); err == nil {
			t.Error("stale complete accepted")
		}
		if err := l.MarkUnknown(ctx, "k", "stale"); err == nil {
			t.Error("stale mark unknown accepted")
		}
		if err := l.Release(ctx, "k", ""); err == nil {
			t.Error("empty fence release accepted")
		}
	})

	t.Run("a completed key replays its result and cannot change", func(t *testing.T) {
		l := newLedger(t)
		claim := begin(t, l, "k")
		stored := done
		stored.LedgerFence, stored.Reason = claim.Fence, "first"
		if err := l.Complete(ctx, "k", claim.Fence, stored); err != nil {
			t.Fatal(err)
		}
		replay := begin(t, l, "k")
		if replay.State != capability.LedgerCompleted || replay.Fence != "" || replay.Result == nil ||
			replay.Result.Outcome != "done" || replay.Result.Reason != "first" || replay.Result.LedgerFence != "" || replay.Result.Err != nil {
			t.Fatalf("replay: %+v", replay)
		}
		if err := l.Complete(ctx, "k", claim.Fence, capability.Result{Outcome: "changed"}); err == nil {
			t.Error("completed result changed")
		}
		if err := l.Release(ctx, "k", claim.Fence); err == nil {
			t.Error("completed key released")
		}
		if err := l.MarkUnknown(ctx, "k", claim.Fence); err == nil {
			t.Error("completed key marked unknown")
		}
	})

	t.Run("a released key can be claimed again", func(t *testing.T) {
		l := newLedger(t)
		claim := begin(t, l, "k")
		if err := l.Release(ctx, "k", claim.Fence); err != nil {
			t.Fatal(err)
		}
		if again := begin(t, l, "k"); again.State != capability.LedgerNew || again.Fence == "" {
			t.Fatalf("released key: %+v", again)
		}
	})

	t.Run("an unknown key blocks until its holder resolves it", func(t *testing.T) {
		l := newLedger(t)
		claim := begin(t, l, "k")
		if err := l.MarkUnknown(ctx, "k", claim.Fence); err != nil {
			t.Fatal(err)
		}
		if blocked := begin(t, l, "k"); blocked.State != capability.LedgerUnknown || blocked.Fence != "" {
			t.Fatalf("unknown key: %+v", blocked)
		}
		if err := l.Complete(ctx, "k", claim.Fence, done); err != nil {
			t.Fatalf("holder completes an unknown key: %v", err)
		}
	})

	t.Run("only an unknown key is reclaimed, and each reclaim supersedes the last", func(t *testing.T) {
		l := newLedger(t)
		if _, err := l.ReclaimUnknown(ctx, "missing"); err == nil {
			t.Error("missing key reclaimed")
		}
		claim := begin(t, l, "k")
		if _, err := l.ReclaimUnknown(ctx, "k"); err == nil {
			t.Error("in-flight key reclaimed")
		}
		if err := l.MarkUnknown(ctx, "k", claim.Fence); err != nil {
			t.Fatal(err)
		}
		first, err := l.ReclaimUnknown(ctx, "k")
		if err != nil || first == "" || first == claim.Fence {
			t.Fatalf("reclaim: %q %v", first, err)
		}
		if blocked := begin(t, l, "k"); blocked.State != capability.LedgerUnknown || blocked.Fence != "" {
			t.Fatalf("reclaimed key must stay unknown to Begin: %+v", blocked)
		}
		second, err := l.ReclaimUnknown(ctx, "k")
		if err != nil || second == "" || second == first {
			t.Fatalf("second reclaim: %q %v", second, err)
		}
		for _, stale := range []string{claim.Fence, first} {
			if err := l.Release(ctx, "k", stale); err == nil {
				t.Fatalf("superseded fence %q released the key", stale)
			}
		}
		if err := l.Complete(ctx, "k", second, done); err != nil {
			t.Fatalf("reconciler completes with its fence: %v", err)
		}
		if _, err := l.ReclaimUnknown(ctx, "k"); err == nil {
			t.Error("completed key reclaimed")
		}
	})
}

func begin(t *testing.T, l capability.Ledger, key string) capability.LedgerEntry {
	t.Helper()
	e, err := l.Begin(context.Background(), key)
	if err != nil {
		t.Fatalf("begin %s: %v", key, err)
	}
	return e
}
