package keel

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nauticana/keel/common"
	"github.com/nauticana/keel/guard"
	kmodel "github.com/nauticana/keel/model"
	"github.com/nauticana/keel/port"

	"github.com/nauticana/charter/sdk/authority"
	"github.com/nauticana/charter/sdk/capability"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/identity"
	"github.com/nauticana/charter/sdk/model"
)

const (
	ns    = "harbor.example"
	agent = "AGENT-ORDER-EXCEPTION-COORDINATOR"
)

var at = time.Date(2026, 6, 18, 17, 12, 30, 0, time.UTC)

func oauthContext(claims map[string]any) context.Context {
	p := &port.Principal{Subject: "sub-42", Scopes: []string{"orders:write", "orders:read"}, Claims: claims}
	ctx := context.WithValue(context.Background(), common.AuthPrincipal, p)
	ctx = context.WithValue(ctx, common.Subject, p.Subject)
	ctx = context.WithValue(ctx, common.Scopes, strings.Join(p.Scopes, " "))
	ctx = context.WithValue(ctx, common.PartnerID, int64(7))
	return common.WithRequestID(ctx, "aB3dE5fG7hJ9")
}

func agentClaims() map[string]any {
	return map[string]any{DefaultKindClaim: "AgentIdentity", DefaultIDClaim: agent, DefaultUserIDClaim: float64(42)}
}

func harbor(t *testing.T) *corpus.Corpus {
	c, err := corpus.NewDirLoader("../../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSessionAndRuntimeContext(t *testing.T) {
	s, err := SessionFromContext(oauthContext(agentClaims()))
	if err != nil || s.Subject != "sub-42" || s.PartnerID != 7 || !s.HasScope("orders:write") || s.HasScope("admin") || s.RequestID == "" {
		t.Fatalf("oauth session: %+v %v", s, err)
	}
	apiKey := context.WithValue(context.WithValue(context.Background(), common.ApiKeyID, int64(9)), common.Scopes, "a,b")
	if s, err := SessionFromContext(apiKey); err != nil || s.APIKeyID != 9 || len(s.Scopes) != 2 {
		t.Errorf("api-key session: %+v %v", s, err)
	}
	if _, err := SessionFromContext(context.Background()); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("anonymous context accepted: %v", err)
	}
	rc, err := RuntimeContext(oauthContext(nil), model.Ref{ID: "RT-OEC-PROD-01"})
	if err != nil || rc.ExecutionContextID != "aB3dE5fG7hJ9" || rc.RuntimeInstanceID.ID != "RT-OEC-PROD-01" {
		t.Errorf("runtime context: %+v %v", rc, err)
	}
	if _, err := RuntimeContext(context.Background(), model.Ref{ID: "RT"}); !errors.Is(err, ErrNoRequestID) {
		t.Errorf("missing request id accepted: %v", err)
	}
	if _, err := RuntimeContext(common.WithRequestID(context.Background(), "bad id!"), model.Ref{ID: "RT"}); !errors.Is(err, ErrNoRequestID) {
		t.Errorf("non-identifier request id accepted: %v", err)
	}
}

func TestCallerResolvesMappedIdentity(t *testing.T) {
	c := Caller{Identities: identity.NewBaseResolver(harbor(t)), Map: BaseClaimIdentityMap{Namespace: ns}}
	actor, s, err := c.Actor(oauthContext(agentClaims()), at)
	if err != nil || actor.ID != agent || actor.Kind != model.KindAgentIdentity || s.PartnerID != 7 {
		t.Fatalf("actor: %+v %v", actor, err)
	}
	if _, _, err := c.Actor(oauthContext(agentClaims()), time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)); !errors.Is(err, identity.ErrLifecycle) {
		t.Errorf("inactive identity accepted: %v", err)
	}
	if _, _, err := c.Actor(oauthContext(map[string]any{"sub": "x"}), at); !errors.Is(err, ErrUnmapped) {
		t.Errorf("claims without identity accepted: %v", err)
	}
	if _, _, err := c.Actor(context.Background(), at); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("anonymous caller accepted: %v", err)
	}
	if _, _, err := (Caller{}).Actor(oauthContext(agentClaims()), at); !errors.Is(err, ErrUnmapped) {
		t.Errorf("missing identity map: %v", err)
	}
	if _, _, err := (Caller{Map: BaseClaimIdentityMap{Namespace: ns}}).Actor(oauthContext(agentClaims()), at); !errors.Is(err, identity.ErrNoProvider) {
		t.Errorf("missing identity provider: %v", err)
	}
	m := BaseClaimIdentityMap{Namespace: ns}
	if uid, err := m.UserID(oauthContext(agentClaims()), model.ObjectRef{Kind: model.KindAgentIdentity, ID: agent}); err != nil || uid != 42 {
		t.Errorf("user id: %d %v", uid, err)
	}
	if _, err := m.UserID(oauthContext(agentClaims()), model.ObjectRef{Kind: model.KindHumanIdentity, ID: "HUMAN-ALEX-RIVERA"}); !errors.Is(err, ErrUnmapped) {
		t.Errorf("session for another actor accepted: %v", err)
	}
}

type checker map[string]bool

func (c checker) CheckActionPermission(_ context.Context, userID int, authObject, action, scope string) (bool, bool) {
	return c[fmt.Sprintf("%d:%s:%s:%s", userID, authObject, action, scope)], false
}

func TestPermissionGateLayersKeelBehindCharter(t *testing.T) {
	c := harbor(t)
	req := authority.Request{Namespace: ns, EnterpriseID: model.Ref{ID: "ENT-HARBOR"}, Actor: model.ObjectRef{Kind: model.KindAgentIdentity, ID: agent},
		CapabilityID: model.Ref{ID: "CAP-RESERVE-ORDER-STOCK"}, ResourceScope: "order exceptions assigned to the agent", OrganizationalContext: "OU-SALES-OPERATIONS", At: at,
		Measures: map[string]authority.Measure{"reservation-value": {Value: int64(1850000), Currency: "USD", CurrencyExponent: 2}, "reservation-duration": {Value: 48, Unit: "hour"}}}
	capabilityKey := corpus.KeyOf(ns, req.CapabilityID)
	perm := Permission{AuthObject: "RESERVATION", Action: "CREATE", Scope: "*"}
	gate := &PermissionGate{Charter: &authority.AbstractEvaluator{Source: authority.NewDocumentGrantSource(c)}, Identities: BaseClaimIdentityMap{Namespace: ns},
		Keel: checker{"42:RESERVATION:CREATE:*": true}, Permissions: map[corpus.DocumentKey]Permission{capabilityKey: perm}}
	ctx := oauthContext(agentClaims())
	if d := gate.Evaluate(ctx, req); d.Result != authority.Allowed || d.GrantRef == nil || !strings.Contains(d.Reason, "keel permission") {
		t.Errorf("both allow: %+v", d)
	}
	gate.Keel = checker{}
	if d := gate.Evaluate(ctx, req); d.Result != authority.Denied || d.GrantRef == nil {
		t.Errorf("keel denial must deny and keep the grant reference: %+v", d)
	}
	gate.Keel = checker{"42:RESERVATION:CREATE:*": true}
	over := req
	over.Measures = map[string]authority.Measure{"reservation-value": {Value: int64(9900000), Currency: "USD", CurrencyExponent: 2}, "reservation-duration": {Value: 48, Unit: "hour"}}
	if d := gate.Evaluate(ctx, over); d.Result != authority.Denied {
		t.Errorf("keel must not widen a Charter denial: %+v", d)
	}
	gate.Permissions = nil
	if d := gate.Evaluate(ctx, req); d.Result != authority.Denied {
		t.Errorf("unmapped capability must fail closed: %+v", d)
	}
	gate.Permissions = map[corpus.DocumentKey]Permission{capabilityKey: perm}
	if d := gate.Evaluate(context.Background(), req); d.Result != authority.Error {
		t.Errorf("anonymous context must not resolve a user: %+v", d)
	}
}

type querier map[string]*kmodel.QueryResult

func (q querier) Query(_ context.Context, name string, _ ...any) (*kmodel.QueryResult, error) {
	if r, ok := q[name]; ok {
		return r, nil
	}
	return nil, errors.New("unknown query " + name)
}
func (querier) GenID() int64 { return 1 }

type fakeInvoker struct {
	calls  int
	result capability.Result
}

func (f *fakeInvoker) Invoke(context.Context, capability.Invocation) capability.Result {
	f.calls++
	return f.result
}

func TestGuardedInvoker(t *testing.T) {
	inv := capability.Invocation{Namespace: ns, CapabilityID: model.Ref{ID: "CAP-RESERVE-ORDER-STOCK"}, IdempotencyKey: "IDEMP-1", At: at}
	ctx := oauthContext(agentClaims())
	next := &fakeInvoker{result: capability.Result{Status: capability.StatusExecuted}}
	rows := func(v any) *kmodel.QueryResult {
		return &kmodel.QueryResult{Columns: []string{"c"}, Rows: [][]any{{v}}}
	}
	guards := guard.NewGuardChain(guard.NewDuplicateGuard("dup", 5*time.Minute), guard.NewMaxCountGuard("rate", "daily rate", 3, 24*time.Hour))
	g := &GuardedInvoker{Next: next, Guards: guards, Querier: querier{"dup": {}, "rate": rows(int64(1))}}
	if res := g.Invoke(ctx, inv); res.Status != capability.StatusExecuted || next.calls != 1 {
		t.Errorf("passing guards: %+v calls=%d", res, next.calls)
	}
	g.Querier = querier{"dup": rows(int64(77)), "rate": rows(int64(1))}
	if res := g.Invoke(ctx, inv); res.Status != capability.StatusUnknown || res.Requirement != "CHR-SEC-008" || !strings.Contains(res.Reason, "77") || next.calls != 1 {
		t.Errorf("duplicate in flight: %+v calls=%d", res, next.calls)
	}
	g.Querier = querier{"dup": {}, "rate": rows(int64(3))}
	if res := g.Invoke(ctx, inv); res.Status != capability.StatusDenied || res.Requirement != "CHR-AGENT-004" || next.calls != 1 {
		t.Errorf("rate limit: %+v calls=%d", res, next.calls)
	}
	g.Querier = querier{}
	if res := g.Invoke(ctx, inv); res.Status != capability.StatusDenied || res.Requirement != "CHR-SEC-007" {
		t.Errorf("guard infrastructure failure must fail closed: %+v", res)
	}
	if res := g.Invoke(context.Background(), inv); res.Status != capability.StatusDenied || res.Requirement != "CHR-SEC-001" {
		t.Errorf("anonymous invocation: %+v", res)
	}
	subjects := inv
	subjects.IdempotencyKey, subjects.SubjectRefs = "", []model.ObjectRef{{Kind: model.KindProcessInstance, ID: "PI-1"}}
	if DedupKey(inv) == DedupKey(subjects) || !strings.HasPrefix(DedupKey(subjects), "harbor.example:CAP-RESERVE-ORDER-STOCK|") {
		t.Errorf("dedup keys: %q %q", DedupKey(inv), DedupKey(subjects))
	}
}

// memoryLogger is a queryable port.TableLogger; the file logger in keel cannot answer FindChanges.
type memoryLogger struct {
	rows        []*kmodel.TableChangeLog
	writeOnly   bool
	lastPartner int64
	lastOwner   int
}

func (l *memoryLogger) Init() error { return nil }
func (l *memoryLogger) LogChange(_ context.Context, c *kmodel.TableChangeLog) error {
	c.ID = int64(len(l.rows) + 1)
	l.rows = append(l.rows, c)
	return nil
}
func (l *memoryLogger) GetChange(_ context.Context, id int64, _ int64, _ int) (*kmodel.TableChangeLog, error) {
	return l.rows[id-1], nil
}
func (l *memoryLogger) FindChanges(_ context.Context, filter port.ChangeFilter, partnerID int64, ownerID int) ([]*kmodel.TableChangeLog, error) {
	if l.writeOnly {
		return nil, errors.New("FindChanges is not implemented")
	}
	l.lastPartner, l.lastOwner = partnerID, ownerID
	var out []*kmodel.TableChangeLog
	for _, r := range l.rows {
		if r.InScope(partnerID, ownerID) && r.TableName == filter.TableName && r.Action == filter.Action && (filter.RecordKey == "" || r.RecordKey == filter.RecordKey) {
			out = append(out, r)
		}
	}
	return out, nil
}
func (l *memoryLogger) Close() {}

func record(id string, supersedes *model.Ref) model.EvidenceRecord {
	r := model.EvidenceRecord{Category: model.CategoryObservedFact, Content: "40 units", RecordedAt: at, Supersedes: supersedes}
	r.CharterSpecVersion, r.Namespace, r.ID = "draft", ns, id
	r.EnterpriseID = &model.Ref{ID: "ENT-HARBOR"}
	return r
}

func TestTableLogStoreIsAppendOnlyAndVerifiable(t *testing.T) {
	ctx := context.Background()
	logger := &memoryLogger{}
	store := &TableLogStore{Logger: logger, PartnerID: 7, OwnerUserID: 42, CreatedBy: 42}
	sink := &evidence.AbstractSink{Store: store, Digester: evidence.BaseSHA256Digester{}}
	if err := sink.Record(ctx, record("EVR-1", nil)); err != nil {
		t.Fatal(err)
	}
	if err := sink.Record(ctx, record("EVR-1", nil)); !errors.Is(err, evidence.ErrDuplicate) {
		t.Errorf("duplicate appended: %v", err)
	}
	if err := sink.Record(ctx, record("EVR-2", &model.Ref{ID: "EVR-1"})); err != nil {
		t.Fatal(err)
	}
	if logger.rows[0].TableName != "charter_evidencerecord" || logger.rows[0].RecordKey != "harbor.example:EVR-1" || logger.rows[0].PartnerID != 7 || logger.rows[0].OwnerUserID != 42 || logger.rows[0].CreatedBy != 42 || logger.rows[0].Action != "I" {
		t.Errorf("change row: %+v", logger.rows[0])
	}
	if logger.lastPartner != 7 || logger.lastOwner != 42 {
		t.Errorf("read scope: partner=%d owner=%d", logger.lastPartner, logger.lastOwner)
	}
	bundle := model.EvidenceBundle{Subject: model.ObjectRef{Kind: model.KindProcessInstance, ID: "PI-1"}, RecordIDs: []model.Ref{{ID: "EVR-1"}, {ID: "EVR-2"}}, AssuranceProfile: "standard"}
	bundle.CharterSpecVersion, bundle.Namespace, bundle.ID = "draft", ns, "EVID-1"
	bundle.EnterpriseID = &model.Ref{ID: "ENT-HARBOR"}
	if err := sink.Bundle(ctx, bundle); err != nil {
		t.Fatal(err)
	}
	if err := (evidence.Verifier{Source: store, Digester: evidence.BaseSHA256Digester{}}).Verify(ctx, ns, model.Ref{ID: "EVID-1"}); err != nil {
		t.Errorf("verify: %v", err)
	}
	if records, err := evidence.NewBaseProvider(store).Records(ctx); err != nil || len(records) != 2 || records[1].Supersedes == nil {
		t.Errorf("records: %v %v", records, err)
	}
	if lineage, err := (evidence.Queries{Provider: evidence.NewBaseProvider(store)}).Lineage(ctx, ns, model.Ref{ID: "EVR-2"}); err != nil || len(lineage) != 2 {
		t.Errorf("lineage: %v %v", lineage, err)
	}
	writeOnly := &memoryLogger{writeOnly: true}
	if err := (&evidence.AbstractSink{Store: &TableLogStore{Logger: writeOnly, PartnerID: 7, OwnerUserID: 42}, Digester: evidence.BaseSHA256Digester{}}).Record(ctx, record("EVR-3", nil)); err == nil || len(writeOnly.rows) != 0 {
		t.Errorf("write-only logger must fail closed: %v rows=%d", err, len(writeOnly.rows))
	}
}

type publisher struct {
	topic string
	data  []byte
	attrs map[string]string
	err   error
}

func (p *publisher) Publish(_ context.Context, topic string, data []byte, attrs map[string]string) error {
	p.topic, p.data, p.attrs = topic, data, attrs
	return p.err
}
func (p *publisher) Close() error { return nil }

func TestPublishingSink(t *testing.T) {
	ctx := oauthContext(nil)
	if err := (&PublishingSink{}).Record(ctx, record("EVR-0", nil)); !errors.Is(err, evidence.ErrNoStore) {
		t.Errorf("incomplete publishing sink: %v", err)
	}
	memory := evidence.NewBaseMemorySink()
	pub := &publisher{}
	sink := &PublishingSink{Next: memory, Publisher: pub, Topic: "charter.evidence"}
	if err := sink.Record(ctx, record("EVR-1", nil)); err != nil {
		t.Fatal(err)
	}
	if pub.topic != "charter.evidence" || pub.attrs["kind"] != "EvidenceRecord" || pub.attrs["id"] != "EVR-1" || pub.attrs["request_id"] != "aB3dE5fG7hJ9" || !strings.Contains(string(pub.data), `"kind":"EvidenceRecord"`) {
		t.Errorf("published event: %s %v", pub.topic, pub.attrs)
	}
	pub.err = errors.New("broker down")
	err := sink.Record(ctx, record("EVR-2", nil))
	if !errors.Is(err, ErrPublish) {
		t.Errorf("publish failure: %v", err)
	}
	if _, err := memory.Store.Fetch(ctx, corpus.DocumentKey{Namespace: ns, ID: "EVR-2"}); err != nil {
		t.Errorf("record must be appended even when publishing fails: %v", err)
	}
	if err := sink.Record(ctx, record("EVR-2", nil)); !errors.Is(err, evidence.ErrDuplicate) {
		t.Errorf("append failure must be returned unchanged: %v", err)
	}
}

type generator struct{ n int64 }

func (g *generator) NextID() int64 { g.n++; return 7000000000000 + g.n }

type metrics struct{ last port.MetricMeasurement }

func (m *metrics) RecordMetric(_ context.Context, mm port.MetricMeasurement) error {
	m.last = mm
	return nil
}

func TestIDsAndMetrics(t *testing.T) {
	ids := &BigintIDs{Generator: &generator{}}
	if id := ids.NewID(model.KindActionRecord); id != "ACT-7000000000001" {
		t.Errorf("id: %s", id)
	}
	if id := ids.NewID(model.KindEvidenceRecord); !strings.HasPrefix(id, "EVIDENCERECORD-") {
		t.Errorf("id: %s", id)
	}
	rec := &metrics{}
	m := &MetricsInvoker{Next: &fakeInvoker{result: capability.Result{Status: capability.StatusDenied, Requirement: "CHR-AUTH-002"}}, Metrics: rec}
	res := m.Invoke(context.Background(), capability.Invocation{CapabilityID: model.Ref{ID: "CAP-X"}})
	if res.Status != capability.StatusDenied || rec.last.Name != DefaultInvocationMetric || rec.last.Labels["status"] != "denied" || rec.last.Labels["requirement"] != "CHR-AUTH-002" || rec.last.Labels["capability"] != "CAP-X" {
		t.Errorf("metric: %+v", rec.last)
	}
	if res := (&MetricsInvoker{}).Invoke(context.Background(), capability.Invocation{}); res.Status != capability.StatusDenied || res.Requirement != "CHR-AUTH-010" {
		t.Errorf("incomplete metrics invoker: %+v", res)
	}
}
