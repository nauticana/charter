package authority

import (
	"context"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func TestApprovalGate(t *testing.T) {
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	gate := &AbstractApprovalGate{Source: NewDocumentApprovalSource(c)}
	base := ApprovalRequest{
		Namespace: "harbor.example", EnterpriseID: model.Ref{ID: "ENT-HARBOR"},
		Actor:                model.ObjectRef{Kind: model.KindAgentIdentity, ID: "AGENT-ORDER-EXCEPTION-COORDINATOR"},
		ApprovedAction:       "CAP-APPROVE-CREDIT-EXCEPTION",
		MaterialInputsDigest: "sha256:4f1d2a9c7b3e5d6f8a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f",
		SubjectRefs:          []model.ObjectRef{{Kind: model.KindProcessInstance, ID: "PROCINST-OE-2026-0042"}},
		At:                   time.Date(2026, 6, 18, 17, 12, 30, 0, time.UTC),
		Measures:             map[string]Measure{"credit-exposure": {Value: int64(1850000), Currency: "USD", CurrencyExponent: 2}},
	}
	cases := []struct {
		name   string
		mutate func(*ApprovalRequest)
		want   ApprovalResult
	}{
		{"bound and valid", func(*ApprovalRequest) {}, Approved},
		{"expired", func(r *ApprovalRequest) { r.At = time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC) }, ApprovalStale},
		{"before issue", func(r *ApprovalRequest) { r.At = time.Date(2026, 6, 18, 15, 0, 0, 0, time.UTC) }, ApprovalStale},
		{"inputs changed", func(r *ApprovalRequest) { r.MaterialInputsDigest = "sha256:changed" }, NoApproval},
		{"other subject", func(r *ApprovalRequest) {
			r.SubjectRefs = []model.ObjectRef{{Kind: model.KindProcessInstance, ID: "PROCINST-OTHER"}}
		}, NoApproval},
		{"over approved limit", func(r *ApprovalRequest) {
			r.Measures = map[string]Measure{"credit-exposure": {Value: int64(2000000), Currency: "USD", CurrencyExponent: 2}}
		}, NoApproval},
		{"other action", func(r *ApprovalRequest) { r.ApprovedAction = "CAP-RESERVE-ORDER-STOCK" }, NoApproval},
	}
	for _, tc := range cases {
		req := base
		tc.mutate(&req)
		d := gate.Evaluate(context.Background(), req)
		if d.Result != tc.want {
			t.Errorf("%s: got %s (%s), want %s", tc.name, d.Result, d.Reason, tc.want)
		} else if tc.want == Approved && (d.ApprovalRef == nil || d.ApprovalRef.ID != "APPR-OE-2026-0042-CREDIT-01" || d.ApprovalRef.Namespace != "harbor.example") {
			t.Errorf("%s: approved without approval reference", tc.name)
		}
	}
	if d := (&AbstractApprovalGate{}).Evaluate(context.Background(), base); d.Result != ApprovalError {
		t.Error("gate without source must fail closed")
	}
	grants := NewDocumentGrantSource(c)
	if gs, err := grants.Grants(context.Background(), Request{Namespace: "harbor.example", EnterpriseID: model.Ref{ID: "ENT-HARBOR"}, Actor: base.Actor, CapabilityID: model.Ref{ID: "CAP-RESERVE-ORDER-STOCK"}}); err != nil || len(gs) != 1 {
		t.Errorf("document grant source: %v %v", gs, err)
	}
}
