package keel

import (
	"context"
	"errors"
	"fmt"

	"github.com/nauticana/keel/common"
	"github.com/nauticana/keel/guard"

	"github.com/nauticana/charter/sdk/authority"
	"github.com/nauticana/charter/sdk/capability"
	"github.com/nauticana/charter/sdk/corpus"
)

// GuardedInvoker runs keel trust guards (duplicate, rate, age) before the governed pipeline; a refusal fails closed
// before authority is evaluated, so nothing external is called and no ledger entry is made (CHR-SEC-008, CHR-AGENT-004).
type GuardedInvoker struct {
	Next    capability.Invoker
	Guards  guard.TrustGuard
	Querier guard.GuardQuerier
}

var _ capability.Invoker = (*GuardedInvoker)(nil)

func (g *GuardedInvoker) Invoke(ctx context.Context, inv capability.Invocation) capability.Result {
	if g.Next == nil || g.Guards == nil || g.Querier == nil {
		return capability.Result{Status: capability.StatusDenied, Reason: "guarded invoker is not fully composed", Requirement: "CHR-AUTH-010"}
	}
	s, err := common.CallerSessionFromContext(ctx)
	if err != nil {
		return capability.Result{Status: capability.StatusDenied, Reason: err.Error(), Requirement: "CHR-SEC-001", Err: err}
	}
	in := guard.GuardInput{PartnerID: s.PartnerID, APIKeyID: s.APIKeyID, DedupKey: DedupKey(inv), Now: inv.At}
	err = g.Guards.Check(ctx, g.Querier, in)
	var dup *guard.DuplicateError
	switch {
	case err == nil:
		return g.Next.Invoke(ctx, inv)
	case errors.As(err, &dup):
		return capability.Result{Status: capability.StatusUnknown, Requirement: "CHR-SEC-008", Err: err,
			Reason: fmt.Sprintf("equivalent request %d is in flight or was recently satisfied; reconcile before retrying", dup.ExistingID)}
	case errors.Is(err, guard.ErrGuardRejected):
		return capability.Result{Status: capability.StatusDenied, Reason: err.Error(), Requirement: "CHR-AGENT-004", Err: err}
	}
	return capability.Result{Status: capability.StatusDenied, Reason: "trust guard: " + err.Error(), Requirement: "CHR-SEC-007", Err: err}
}

// DedupKey identifies equivalent invocations: the capability plus the idempotency key, or plus the action subjects when no key is set.
func DedupKey(inv capability.Invocation) string {
	key := inv.IdempotencyKey
	if key == "" {
		key = authority.SubjectScope(inv.Namespace, inv.SubjectRefs)
	}
	return corpus.KeyOf(inv.Namespace, inv.CapabilityID).String() + "|" + key
}
