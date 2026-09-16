package capability_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/agent"
	"github.com/nauticana/charter/sdk/authority"
	"github.com/nauticana/charter/sdk/binding"
	"github.com/nauticana/charter/sdk/capability"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/identity"
	"github.com/nauticana/charter/sdk/information"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/organization"
	"github.com/nauticana/charter/sdk/validate"
)

type fakeTransport struct {
	calls int
	resp  binding.Response
	err   error
}

type malformedLedger struct{}

func (malformedLedger) Begin(context.Context, string) (capability.LedgerEntry, error) {
	return capability.LedgerEntry{State: capability.LedgerCompleted}, nil
}
func (malformedLedger) Complete(context.Context, string, string, capability.Result) error { return nil }
func (malformedLedger) Release(context.Context, string, string) error                     { return nil }
func (malformedLedger) MarkUnknown(context.Context, string, string) error                 { return nil }

type unfencedLedger struct{ malformedLedger }

func (unfencedLedger) Begin(context.Context, string) (capability.LedgerEntry, error) {
	return capability.LedgerEntry{State: capability.LedgerNew}, nil
}

type invalidStateLedger struct{ malformedLedger }

func (invalidStateLedger) Begin(context.Context, string) (capability.LedgerEntry, error) {
	return capability.LedgerEntry{State: capability.LedgerState("corrupt"), Fence: "must-not-leak"}, nil
}

func (f *fakeTransport) Execute(context.Context, binding.Request) (binding.Response, error) {
	f.calls++
	return f.resp, f.err
}

type harness struct {
	t          *testing.T
	corpus     *corpus.Corpus
	sink       *evidence.BaseMemorySink
	transport  *fakeTransport
	invoker    *capability.AbstractInvoker
	structural *validate.Structural
}

const digest = "sha256:4f1d2a9c7b3e5d6f8a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f"

var at = time.Date(2026, 6, 18, 17, 12, 30, 0, time.UTC)

func newHarness(t *testing.T) *harness {
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		`{"charterSpecVersion":"1.0.0","namespace":"harbor.example","kind":"CapabilityBinding","id":"BIND-READ-1","bindingVersion":"1","systemProfileId":"SYSPROFILE-HARBOR-S4-2602","capabilityId":"CAP-READ-ORDER-EXCEPTION","operation":"GET /orders","featureSupport":[{"feature":"order-read","support":"supported"}],"mappings":{"inputs":"i","outputs":"o","businessErrors":"b","authorityChecks":"a","idempotency":"none","evidence":"e"}}`,
		`{"charterSpecVersion":"1.0.0","namespace":"harbor.example","kind":"HumanIdentity","id":"HUMAN-TEST","enterpriseId":"ENT-HARBOR","name":"Test Approver","lifecycleState":"active","lifecycleHistory":[{"state":"active","effectiveAt":"2026-01-01T00:00:00Z"}]}`,
		`{"charterSpecVersion":"1.0.0","namespace":"harbor.example","kind":"CapabilityBinding","id":"BIND-APPROVE-1","bindingVersion":"1","systemProfileId":"SYSPROFILE-HARBOR-S4-2602","capabilityId":"CAP-APPROVE-CREDIT-EXCEPTION","operation":"POST /credit-decisions","featureSupport":[{"feature":"credit-decision","support":"supported"}],"mappings":{"inputs":"i","outputs":"o","businessErrors":"b","authorityChecks":"a","idempotency":"digest","evidence":"e"}}`,
	} {
		d, err := (corpus.Parser{}).Parse([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if err := c.Add(d); err != nil {
			t.Fatal(err)
		}
	}
	structural, err := validate.NewStructural()
	if err != nil {
		t.Fatal(err)
	}
	sink := evidence.NewBaseMemorySink()
	transport := &fakeTransport{resp: binding.Response{Outcome: "reserved", ExternalReference: "0000088421"}}
	return &harness{t: t, corpus: c, sink: sink, transport: transport, structural: structural, invoker: &capability.AbstractInvoker{
		Catalog:    capability.NewBaseCatalog(c),
		Identities: identity.NewBaseResolver(c),
		Authority:  &authority.AbstractEvaluator{Source: authority.NewDocumentGrantSource(c)},
		Approvals:  &authority.AbstractApprovalGate{Source: authority.NewDocumentApprovalSource(c)},
		Sod:        capability.NewBaseSodChecker(c, evidence.NewBaseProvider(sink.Store)),
		Bindings:   binding.NewBaseProvider(c),
		Transport:  transport,
		Ledger:     capability.NewBaseMemoryLedger(),
		Evidence:   sink,
		IDs:        &capability.BaseCounterIDs{},
	}}
}

func reserve() capability.Invocation {
	return capability.Invocation{
		Namespace: "harbor.example", EnterpriseID: model.Ref{ID: "ENT-HARBOR"},
		Actor:        model.ObjectRef{Kind: model.KindAgentIdentity, ID: "AGENT-ORDER-EXCEPTION-COORDINATOR"},
		Runtime:      model.RuntimeContext{RuntimeInstanceID: model.Ref{ID: "RT-OEC-PROD-01"}, ExecutionContextID: "EXEC-OE-0042-05"},
		AssignmentID: &model.Ref{ID: "ASGN-OEC-ORDER-EXCEPTION-SUPPORT"},
		CapabilityID: model.Ref{ID: "CAP-RESERVE-ORDER-STOCK"}, ContractVersion: "1", RequiredFeatures: []string{"reservation-create"},
		ResourceScope: "order exceptions assigned to the agent", OrganizationalContext: "OU-SALES-OPERATIONS",
		Measures: map[string]authority.Measure{
			"reservation-value":    {Value: int64(1850000), Currency: "USD", CurrencyExponent: 2},
			"reservation-duration": {Value: 48, Unit: "hour"},
			"credit-exposure":      {Value: int64(1850000), Currency: "USD", CurrencyExponent: 2},
		},
		ApprovedAction: "CAP-APPROVE-CREDIT-EXCEPTION", MaterialInputsDigest: digest,
		SubjectRefs:    []model.ObjectRef{{Kind: model.KindProcessInstance, ID: "PROCINST-OE-2026-0042"}},
		IdempotencyKey: "IDEMP-0042-1", At: at,
	}
}

// recorded asserts the result carries an appended, schema-valid ActionRecord.
func (h *harness) recorded(res capability.Result) model.ActionRecord {
	h.t.Helper()
	if res.Action == nil {
		h.t.Fatalf("no action record for %s (%s)", res.Status, res.Reason)
	}
	d, err := h.sink.Store.Fetch(context.Background(), corpus.DocumentKey{Namespace: res.Action.Namespace, ID: res.Action.ID})
	if err != nil {
		h.t.Fatalf("record not appended: %v", err)
	}
	for _, f := range h.structural.ValidateDocument(d) {
		h.t.Errorf("record %s: %s %s", d.ID, f.Path, f.Message)
	}
	if res.Err != nil {
		h.t.Errorf("result error: %v", res.Err)
	}
	return *res.Action
}

func TestGovernedExecutionAndReplay(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	res := h.invoker.Invoke(ctx, reserve())
	if res.Status != capability.StatusExecuted || res.Outcome != "reserved" || res.ExternalReference != "0000088421" || res.Binding == nil || res.Binding.ID != "BIND-S4-RESERVE-ORDER-STOCK-1" {
		t.Fatalf("execution: %+v", res)
	}
	if res.LedgerFence != "" {
		t.Errorf("completed execution leaked ledger fence %q", res.LedgerFence)
	}
	rec := h.recorded(res)
	if len(rec.AuthorityEvaluations) != 1 || rec.AuthorityEvaluations[0].Result != model.AuthorityAllowed || rec.AuthorityEvaluations[0].AuthorityGrantID.ID != "AUTH-OEC-STOCK-RESERVATION-2026" {
		t.Errorf("authority evidence: %+v", rec.AuthorityEvaluations)
	}
	if len(rec.ApprovalEvaluations) != 1 || rec.ApprovalEvaluations[0].Result != model.ApprovalApproved || rec.ApprovalEvaluations[0].ApprovalID.ID != "APPR-OE-2026-0042-CREDIT-01" || len(rec.ApprovalIDs) != 1 {
		t.Errorf("approval evidence: %+v", rec.ApprovalEvaluations)
	}
	if rec.OperationClass != model.OperationExecute || rec.Outcome != "reserved" || rec.EnterpriseID == nil || rec.ActionTime != at {
		t.Errorf("record: %+v", rec)
	}
	replay := h.invoker.Invoke(ctx, reserve())
	if replay.Status != capability.StatusExecuted || replay.Action.ID != rec.ID || h.transport.calls != 1 {
		t.Errorf("replay must return the prior result without executing again: %+v calls=%d", replay, h.transport.calls)
	}
}

func TestDelegationChainBoundsReachAuthority(t *testing.T) {
	h := newHarness(t)
	inv := reserve()
	inv.AuthorityChain = model.AuthorityChain{{
		GrantRef: model.Ref{ID: "GRANT-EXPIRED"}, Delegator: model.ObjectRef{Kind: model.KindHumanIdentity, ID: "HUMAN-ALEX-RIVERA"},
		Validity: &model.Validity{From: "2025-01-01", To: "2025-12-31"},
	}}
	res := h.invoker.Invoke(context.Background(), inv)
	if res.Status != capability.StatusDenied || res.Requirement != "CHR-AUTH-010" || !strings.Contains(res.Reason, "delegation hop") || h.transport.calls != 0 {
		t.Errorf("expired delegation reached execution: %+v calls=%d", res, h.transport.calls)
	}
	h.recorded(res)
}

func TestDelegationChainCanRequireApproval(t *testing.T) {
	h := newHarness(t)
	h.invoker.ReadWithoutGrant = true
	inv := reserve()
	inv.CapabilityID, inv.RequiredFeatures, inv.IdempotencyKey = model.Ref{ID: "CAP-READ-ORDER-EXCEPTION"}, nil, ""
	inv.ApprovedAction = ""
	inv.AuthorityChain = model.AuthorityChain{{
		GrantRef: model.Ref{ID: "GRANT-APPROVAL-BOUND"}, Delegator: model.ObjectRef{Kind: model.KindHumanIdentity, ID: "HUMAN-ALEX-RIVERA"},
		Validity: &model.Validity{From: "2026-01-01"}, ApprovalRequired: true,
	}}
	res := h.invoker.Invoke(context.Background(), inv)
	if res.Status != capability.StatusDenied || res.Requirement != "CHR-AUTH-009" || !strings.Contains(res.Reason, "approval") || h.transport.calls != 0 {
		t.Errorf("delegation approval bound ignored: %+v calls=%d", res, h.transport.calls)
	}
	h.recorded(res)
}

func TestDenialsFailClosedAndAreRecorded(t *testing.T) {
	cases := []struct {
		name        string
		mutate      func(*capability.Invocation)
		requirement string
	}{
		{"over grant limit", func(i *capability.Invocation) {
			i.Measures["reservation-value"] = authority.Measure{Value: int64(2700000), Currency: "USD", CurrencyExponent: 2}
		}, "CHR-AUTH-010"},
		{"grant expired", func(i *capability.Invocation) { i.At = time.Date(2027, 1, 5, 0, 0, 0, 0, time.UTC) }, "CHR-AUTH-010"},
		{"actor not yet active", func(i *capability.Invocation) { i.At = time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC) }, "CHR-ID-005"},
		{"no grant", func(i *capability.Invocation) { i.CapabilityID = model.Ref{ID: "CAP-REQUEST-EXCEPTION-APPROVAL"} }, "CHR-AUTH-002"},
		{"stale approval", func(i *capability.Invocation) { i.At = time.Date(2026, 6, 19, 17, 0, 0, 0, time.UTC) }, "CHR-AUTH-009"},
		{"inputs changed since approval", func(i *capability.Invocation) { i.MaterialInputsDigest = "sha256:changed" }, "CHR-AUTH-009"},
		{"incompatible contract version", func(i *capability.Invocation) { i.ContractVersion = "2" }, "CHR-CAP-008"},
		{"unsupported binding feature", func(i *capability.Invocation) { i.RequiredFeatures = []string{"reservation-expiry"} }, "CHR-BIND-006"},
		{"unknown system profile", func(i *capability.Invocation) { i.SystemProfileID = &model.Ref{ID: "SYSPROFILE-OTHER"} }, "CHR-BIND-009"},
		{"missing idempotency key", func(i *capability.Invocation) { i.IdempotencyKey = "" }, "CHR-CAP-004"},
	}
	for _, tc := range cases {
		h := newHarness(t)
		inv := reserve()
		tc.mutate(&inv)
		res := h.invoker.Invoke(context.Background(), inv)
		if res.Status != capability.StatusDenied || res.Requirement != tc.requirement || h.transport.calls != 0 {
			t.Errorf("%s: got %s %s (%s) calls=%d, want denied %s", tc.name, res.Status, res.Requirement, res.Reason, h.transport.calls, tc.requirement)
			continue
		}
		if rec := h.recorded(res); rec.Outcome != "denied" {
			t.Errorf("%s: record outcome %s", tc.name, rec.Outcome)
		}
	}
	h := newHarness(t)
	inv := reserve()
	inv.AssignmentID = nil
	if res := h.invoker.Invoke(context.Background(), inv); res.Status != capability.StatusDenied || res.Requirement != "CHR-EVID-001" || res.Action != nil {
		t.Errorf("unattributable invocation: %+v", res)
	}
	if res := (&capability.AbstractInvoker{}).Invoke(context.Background(), reserve()); res.Status != capability.StatusDenied {
		t.Error("incomplete invoker must deny")
	}
}

func TestUnknownOutcomeBlocksRetryUntilReconciled(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	h.transport.err = fmt.Errorf("%w: gateway timeout", binding.ErrOutcomeUnknown)
	res := h.invoker.Invoke(ctx, reserve())
	if res.Status != capability.StatusUnknown || res.Requirement != "CHR-SEC-008" || !errors.Is(res.Err, binding.ErrOutcomeUnknown) {
		t.Fatalf("timeout: %+v", res)
	}
	if rec := h.recorded(capability.Result{Action: res.Action}); rec.Outcome != "unknown" {
		t.Errorf("record outcome %s", rec.Outcome)
	}
	h.transport.err = nil
	retry := h.invoker.Invoke(ctx, reserve())
	if retry.Status != capability.StatusUnknown || h.transport.calls != 1 {
		t.Fatalf("retry before reconciliation must not execute: %+v calls=%d", retry, h.transport.calls)
	}
	reconciled := capability.Result{Status: capability.StatusExecuted, Outcome: "already-reserved", ExternalReference: "0000088421"}
	if err := h.invoker.Ledger.Complete(ctx, "IDEMP-0042-1", "stale", reconciled); !errors.Is(err, capability.ErrLedgerFence) {
		t.Errorf("reconciliation without the fence accepted: %v", err)
	}
	if res.LedgerFence == "" || retry.LedgerFence != "" {
		t.Errorf("fence must reach only the claim holder: %q %q", res.LedgerFence, retry.LedgerFence)
	}
	if err := h.invoker.Ledger.Complete(ctx, "IDEMP-0042-1", res.LedgerFence, reconciled); err != nil {
		t.Fatal(err)
	}
	if after := h.invoker.Invoke(ctx, reserve()); after.Status != capability.StatusExecuted || after.Outcome != "already-reserved" || h.transport.calls != 1 {
		t.Errorf("reconciled result not replayed: %+v", after)
	}

	h = newHarness(t)
	h.transport.err = fmt.Errorf("%w: connection refused", binding.ErrNotExecuted)
	if res := h.invoker.Invoke(ctx, reserve()); res.Status != capability.StatusFailed || res.Requirement != "CHR-SEC-007" {
		t.Fatalf("proven non-execution: %+v", res)
	}
	h.transport.err = nil
	if res := h.invoker.Invoke(ctx, reserve()); res.Status != capability.StatusExecuted || h.transport.calls != 2 {
		t.Errorf("retry after proven non-execution: %+v calls=%d", res, h.transport.calls)
	}
}

func TestCompletedLedgerEntryCannotBeChanged(t *testing.T) {
	ctx := context.Background()
	ledger := capability.NewBaseMemoryLedger()
	entry, err := ledger.Begin(ctx, "IDEMP-COMPLETE")
	if err != nil {
		t.Fatal(err)
	}
	done := capability.Result{Status: capability.StatusExecuted, Outcome: "done", LedgerFence: entry.Fence, Err: errors.New("transient")}
	stored := done
	stored.LedgerFence = ""
	stored.Err = nil
	if err := ledger.Complete(ctx, "IDEMP-COMPLETE", entry.Fence, done); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Complete(ctx, "IDEMP-COMPLETE", entry.Fence, stored); err != nil {
		t.Errorf("identical completion is not idempotent: %v", err)
	}
	if err := ledger.Complete(ctx, "IDEMP-COMPLETE", entry.Fence, capability.Result{Outcome: "changed"}); !errors.Is(err, capability.ErrLedgerTransition) {
		t.Errorf("completed result changed: %v", err)
	}
	if err := ledger.Release(ctx, "IDEMP-COMPLETE", entry.Fence); !errors.Is(err, capability.ErrLedgerTransition) {
		t.Errorf("completed result released: %v", err)
	}
	replay, err := ledger.Begin(ctx, "IDEMP-COMPLETE")
	if err != nil || replay.Result == nil || replay.Result.LedgerFence != "" || replay.Result.Outcome != "done" {
		t.Errorf("replay: %+v %v", replay, err)
	}
}

func TestMalformedLedgerClaimFailsBeforeExecution(t *testing.T) {
	for name, ledger := range map[string]capability.Ledger{"unfenced": unfencedLedger{}, "invalid state": invalidStateLedger{}} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			h.invoker.Ledger = ledger
			res := h.invoker.Invoke(context.Background(), reserve())
			if res.Status != capability.StatusUnknown || res.Requirement != "CHR-SEC-008" || res.LedgerFence != "" || h.transport.calls != 0 {
				t.Errorf("malformed claim reached execution: %+v calls=%d", res, h.transport.calls)
			}
		})
	}
}

func TestMalformedCompletedLedgerEntryFailsClosed(t *testing.T) {
	h := newHarness(t)
	h.invoker.Ledger = malformedLedger{}
	res := h.invoker.Invoke(context.Background(), reserve())
	if res.Status != capability.StatusUnknown || res.Requirement != "CHR-SEC-008" || h.transport.calls != 0 {
		t.Fatalf("malformed completed entry: %+v calls=%d", res, h.transport.calls)
	}
	h.recorded(res)
}

func TestOutcomesAndBusinessErrorsMustBeDeclared(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	h.transport.resp = binding.Response{Outcome: "maybe-reserved"}
	if res := h.invoker.Invoke(ctx, reserve()); res.Status != capability.StatusUnknown || res.Requirement != "CHR-CAP-001" {
		t.Errorf("undeclared outcome: %+v", res)
	}
	h = newHarness(t)
	h.transport.resp = binding.Response{BusinessError: "insufficient eligible stock"}
	res := h.invoker.Invoke(ctx, reserve())
	if res.Status != capability.StatusBusinessError || res.BusinessError != "insufficient eligible stock" {
		t.Errorf("declared business error: %+v", res)
	}
	if rec := h.recorded(res); rec.Outcome != "insufficient eligible stock" {
		t.Errorf("business error outcome: %s", rec.Outcome)
	}
	h = newHarness(t)
	h.transport.resp = binding.Response{BusinessError: "HTTP 500"}
	if res := h.invoker.Invoke(ctx, reserve()); res.Status != capability.StatusUnknown || res.Requirement != "CHR-CAP-005" {
		t.Errorf("undeclared business error: %+v", res)
	}
}

func approve() capability.Invocation {
	return capability.Invocation{
		Namespace: "harbor.example", EnterpriseID: model.Ref{ID: "ENT-HARBOR"},
		Actor:            model.ObjectRef{Kind: model.KindHumanIdentity, ID: "HUMAN-TEST"},
		Runtime:          model.RuntimeContext{RuntimeInstanceID: model.Ref{ID: "RT-OEC-PROD-01"}, ExecutionContextID: "EXEC-OE-0042-04"},
		ResponsibilityID: &model.Ref{ID: "RESP-APPROVE-CREDIT-EXCEPTION"},
		CapabilityID:     model.Ref{ID: "CAP-APPROVE-CREDIT-EXCEPTION"}, ResourceScope: "credit exceptions", OrganizationalContext: "OU-CREDIT-CONTROL",
		SubjectRefs:    []model.ObjectRef{{Kind: model.KindProcessInstance, ID: "PROCINST-OE-2026-0042"}},
		IdempotencyKey: "IDEMP-0042-APPROVE", At: at,
	}
}

func TestSeparationOfDuties(t *testing.T) {
	ctx := context.Background()
	grant := model.AuthorityGrant{Actor: approve().Actor, CapabilityID: approve().CapabilityID, ResourceScope: "credit exceptions", OrganizationalContext: "OU-CREDIT-CONTROL"}
	grant.Namespace, grant.ID, grant.Kind = "harbor.example", "AUTH-ALEX-APPROVE", model.KindAuthorityGrant
	grant.EnterpriseID, grant.Validity = &model.Ref{ID: "ENT-HARBOR"}, &model.Validity{From: "2026-01-01"}
	for _, prepared := range []bool{false, true} {
		h := newHarness(t)
		h.invoker.Authority = &authority.AbstractEvaluator{Source: &authority.BaseGrantSource{Items: []model.AuthorityGrant{grant}}}
		h.transport.resp = binding.Response{Outcome: "approved"}
		if prepared {
			proposal := model.ActionRecord{Actor: approve().Actor, RuntimeContext: approve().Runtime, ResponsibilityID: approve().ResponsibilityID,
				CapabilityID: model.Ref{ID: "CAP-PROPOSE-ORDER-RESOLUTION"}, SubjectRefs: approve().SubjectRefs, OperationClass: model.OperationPropose,
				ActionTime: at.Add(-time.Hour), Outcome: "proposal-prepared", AuthorityEvaluations: []model.AuthorityEvaluation{{Result: model.AuthorityAllowed, Reason: "test"}}}
			proposal.CharterSpecVersion, proposal.Namespace, proposal.ID = "1.0.0", "harbor.example", "ACT-ALEX-PROPOSE"
			if err := h.sink.Action(ctx, proposal); err != nil {
				t.Fatal(err)
			}
		}
		res := h.invoker.Invoke(ctx, approve())
		switch {
		case prepared && (res.Status != capability.StatusDenied || res.Requirement != "CHR-AUTH-007" || res.SodConflict == nil || res.SodConflict.ID != "SOD-PREPARE-APPROVE-CREDIT"):
			t.Errorf("preparer approving: %+v", res)
		case !prepared && (res.Status != capability.StatusExecuted || res.Outcome != "approved"):
			t.Errorf("independent approver: %+v", res)
		}
		h.recorded(res)
	}

	h := newHarness(t)
	delegator := approve().Actor
	proposal := model.ActionRecord{Actor: delegator, RuntimeContext: approve().Runtime, ResponsibilityID: approve().ResponsibilityID,
		CapabilityID: model.Ref{ID: "CAP-PROPOSE-ORDER-RESOLUTION"}, SubjectRefs: approve().SubjectRefs, OperationClass: model.OperationPropose,
		ActionTime: at.Add(-time.Hour), Outcome: "proposal-prepared", AuthorityEvaluations: []model.AuthorityEvaluation{{Result: model.AuthorityAllowed, Reason: "test"}}}
	proposal.CharterSpecVersion, proposal.Namespace, proposal.ID = "1.0.0", "harbor.example", "ACT-DELEGATOR-PROPOSE"
	if err := h.sink.Action(ctx, proposal); err != nil {
		t.Fatal(err)
	}
	delegated := approve()
	delegated.Actor = model.ObjectRef{Kind: model.KindAgentIdentity, ID: delegator.ID}
	delegated.AuthorityChain = model.AuthorityChain{{GrantRef: model.Ref{ID: "GRANT-DELEGATED-APPROVAL"}, Delegator: delegator, Validity: &model.Validity{From: "2026-01-01"}}}
	if ref, conflict, err := h.invoker.Sod.Conflict(ctx, delegated, []model.Ref{{ID: "SOD-PREPARE-APPROVE-CREDIT"}}); err != nil || !conflict || ref.ID != "SOD-PREPARE-APPROVE-CREDIT" {
		t.Errorf("delegator history must participate in separation of duties: %s %t %v", ref.ID, conflict, err)
	}
}

func TestReadWithoutGrantIsExplicit(t *testing.T) {
	ctx := context.Background()
	read := reserve()
	read.CapabilityID, read.RequiredFeatures, read.IdempotencyKey, read.MaterialInputsDigest = model.Ref{ID: "CAP-READ-ORDER-EXCEPTION"}, nil, "", ""
	for _, allowed := range []bool{false, true} {
		h := newHarness(t)
		h.invoker.ReadWithoutGrant = allowed
		h.transport.resp = binding.Response{Outcome: "facts-returned"}
		res := h.invoker.Invoke(ctx, read)
		if !allowed && (res.Status != capability.StatusDenied || res.Requirement != "CHR-AUTH-002") {
			t.Errorf("read without grant: %+v", res)
		}
		if allowed && (res.Status != capability.StatusExecuted || res.Action.AuthorityEvaluations[0].Result != model.AuthorityMissing) {
			t.Errorf("assignment-based read: %+v", res)
		}
		h.recorded(res)
	}
}

func TestVersionPolicyAndScope(t *testing.T) {
	var p capability.BaseVersionPolicy
	for required, offered := range map[string]string{"": "1", "1": "1", "1.2": "1.3", "1.0": "1"} {
		if err := p.Compatible(required, offered); err != nil {
			t.Errorf("%q vs %q: %v", required, offered, err)
		}
	}
	for required, offered := range map[string]string{"2": "1", "1.4": "1.3", "v1": "1"} {
		if err := p.Compatible(required, offered); !errors.Is(err, capability.ErrIncompatibleVersion) {
			t.Errorf("%q vs %q accepted", required, offered)
		}
	}
	a := authority.SubjectScope("t", []model.ObjectRef{{Kind: "B", ID: "2"}, {Kind: "A", ID: "1"}})
	b := authority.SubjectScope("t", []model.ObjectRef{{Kind: "A", ID: "1", Namespace: "t"}, {Kind: "B", ID: "2"}})
	if a != b || a == "" {
		t.Errorf("scope keys differ: %q %q", a, b)
	}
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(`{}`), &raw); err != nil {
		t.Fatal(err)
	}
}

func TestInformationGovernanceGate(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	h.invoker.Information = &information.BaseEvaluator{Provider: information.NewBaseProvider(h.corpus)}
	inv := reserve()
	inv.InformationUses = []capability.InformationUse{{Information: model.Ref{ID: "INFO-CREDIT-STATUS-SUMMARY"}, Purpose: "governed approval and execution"}}
	res := h.invoker.Invoke(ctx, inv)
	if res.Status != capability.StatusExecuted || len(res.Information) != 1 || res.Information[0].Result != information.Allowed {
		t.Fatalf("permitted use: %+v", res)
	}
	rec := h.recorded(res)
	if len(rec.InformationEvaluations) != 1 || rec.InformationEvaluations[0].Result != model.InformationAllowed || rec.InformationEvaluations[0].InformationID.ID != "INFO-CREDIT-STATUS-SUMMARY" || rec.Disposition != model.DispositionExecuted {
		t.Errorf("information evidence: %+v", rec)
	}
	inv.InformationUses[0].Purpose = "marketing analysis"
	inv.IdempotencyKey = "IDEMP-0042-2"
	if res := h.invoker.Invoke(ctx, inv); res.Status != capability.StatusDenied || res.Requirement != "CHR-INFO-007" || h.transport.calls != 1 {
		t.Errorf("undeclared purpose: %+v", res)
	} else {
		h.recorded(res)
	}
	h.invoker.Information = nil
	inv.IdempotencyKey = "IDEMP-0042-3"
	if res := h.invoker.Invoke(ctx, inv); res.Status != capability.StatusDenied || res.Requirement != "CHR-INFO-008" {
		t.Errorf("declared use without evaluator must fail closed: %+v", res)
	}
}

func TestStoppedActionsAreEscalated(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	h.invoker.Escalation = &capability.BaseEscalator{Recipients: &capability.BaseAccountableRecipient{Agents: agent.NewBaseProvider(h.corpus), Organization: organization.NewBaseProvider(h.corpus)}, IDs: h.invoker.IDs}
	if res := h.invoker.Invoke(ctx, reserve()); res.Status != capability.StatusExecuted || res.Escalation != nil || res.Exception != nil {
		t.Fatalf("success must not escalate: %+v", res)
	}
	h = newHarness(t)
	h.invoker.Escalation = &capability.BaseEscalator{Recipients: &capability.BaseAccountableRecipient{Agents: agent.NewBaseProvider(h.corpus), Organization: organization.NewBaseProvider(h.corpus)}, IDs: h.invoker.IDs}
	h.transport.err = fmt.Errorf("%w: timeout", binding.ErrOutcomeUnknown)
	res := h.invoker.Invoke(ctx, reserve())
	if res.Status != capability.StatusUnknown || res.Escalation == nil || res.Exception == nil || res.Err == nil {
		t.Fatalf("unknown outcome: %+v", res)
	}
	provider := evidence.NewBaseProvider(h.sink.Store)
	escalation, err := provider.Escalation(ctx, res.Escalation.Namespace, *res.Escalation)
	if err != nil || escalation.Recipient.ID != "POS-SALES-OPERATIONS-MANAGER" || escalation.Urgency != "high" || !strings.Contains(escalation.RequestedDecision, "IDEMP-0042-1") {
		t.Errorf("escalation: %+v %v", escalation, err)
	}
	exception, err := provider.Exception(ctx, res.Exception.Namespace, *res.Exception)
	if err != nil || exception.Affected.ID != "PROCINST-OE-2026-0042" || exception.EscalationTarget.ID != "POS-SALES-OPERATIONS-MANAGER" || exception.Disposition != "open" {
		t.Errorf("exception: %+v %v", exception, err)
	}
	for _, d := range []*model.Document{mustFetch(t, h, res.Exception), mustFetch(t, h, res.Escalation)} {
		for _, f := range h.structural.ValidateDocument(d) {
			t.Errorf("%s: %s %s", d.ID, f.Path, f.Message)
		}
	}
	human := approve()
	human.IdempotencyKey = "IDEMP-H"
	recipient, err := (&capability.BaseAccountableRecipient{Organization: organization.NewBaseProvider(h.corpus)}).Recipient(ctx, human)
	if err != nil || recipient.ID != "POS-CREDIT-MANAGER" {
		t.Errorf("human recipient via responsibility: %+v %v", recipient, err)
	}
	brokenAgent := reserve()
	brokenAgent.Runtime.RuntimeInstanceID = model.Ref{ID: "RT-MISSING"}
	brokenAgent.ResponsibilityID = &model.Ref{ID: "RESP-APPROVE-CREDIT-EXCEPTION"}
	if _, err := (&capability.BaseAccountableRecipient{Agents: agent.NewBaseProvider(h.corpus), Organization: organization.NewBaseProvider(h.corpus)}).Recipient(ctx, brokenAgent); !errors.Is(err, corpus.ErrNotFound) {
		t.Errorf("agent lookup failure must not fall back to a responsibility recipient: %v", err)
	}
}

func mustFetch(t *testing.T, h *harness, ref *model.Ref) *model.Document {
	t.Helper()
	d, err := h.sink.Store.Fetch(context.Background(), corpus.DocumentKey{Namespace: ref.Namespace, ID: ref.ID})
	if err != nil {
		t.Fatal(err)
	}
	return d
}
