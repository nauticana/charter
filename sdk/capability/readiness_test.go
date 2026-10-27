package capability_test

import (
	"context"
	"slices"
	"testing"

	"github.com/nauticana/charter/sdk/capability"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/information"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/predicate"
)

func requirements(blockers []capability.Blocker) []string {
	out := make([]string, 0, len(blockers))
	for _, b := range blockers {
		out = append(out, b.Requirement)
	}
	return out
}

func TestReadinessReportsWhatTheInvokerWouldDeny(t *testing.T) {
	ctx := context.Background()
	profile := &model.Ref{ID: "SYSPROFILE-HARBOR-S4-2602"}
	reservation := model.Ref{ID: "CAP-RESERVE-ORDER-STOCK"}

	if b := newHarness(t).invoker.Readiness(ctx, "harbor.example", reservation, profile, "reservation-create"); len(b) != 0 {
		t.Fatalf("composed Harbor reservation is not ready: %+v", b)
	}

	t.Run("missing observer is reported before the invoker denies it", func(t *testing.T) {
		h := newHarness(t)
		h.invoker.Observer = nil
		b := h.invoker.Readiness(ctx, "harbor.example", reservation, profile, "reservation-create")
		if len(b) != 1 || b[0].Requirement != "CHR-CAP-010" {
			t.Fatalf("blockers = %+v", b)
		}
		if res := h.invoker.Invoke(ctx, reserve()); res.Status != capability.StatusDenied || res.Requirement != b[0].Requirement {
			t.Fatalf("invocation = %s %s, readiness = %s", res.Status, res.Requirement, b[0].Requirement)
		}
		if h.transport.calls != 0 {
			t.Fatal("transport called without an observer")
		}
	})

	t.Run("composition the contract requires", func(t *testing.T) {
		h := newHarness(t)
		h.invoker.Ledger, h.invoker.Approvals, h.invoker.Sod = nil, nil, nil
		h.amend(func(c *model.CapabilityContract) {
			c.Constraints.ApprovalRequired = true
			c.Constraints.SodConstraintIDs = []model.Ref{{ID: "SOD-ANY"}}
		})
		got := requirements(h.invoker.Readiness(ctx, "harbor.example", reservation, profile, "reservation-create"))
		want := []string{"CHR-AUTH-009", "CHR-AUTH-007", "CHR-SEC-008"}
		if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
			t.Fatalf("blockers = %v, want %v", got, want)
		}
	})

	t.Run("bindings, features, and authority", func(t *testing.T) {
		h := newHarness(t)
		if got := requirements(h.invoker.Readiness(ctx, "harbor.example", reservation, profile, "reservation-cancel")); len(got) != 1 || got[0] != "CHR-BIND-006" {
			t.Fatalf("unsupported feature = %v", got)
		}
		if got := requirements(h.invoker.Readiness(ctx, "harbor.example", reservation, &model.Ref{ID: "SYSPROFILE-OTHER"})); len(got) != 2 || got[0] != "CHR-BIND-009" || got[1] != "CHR-BIND-006" {
			t.Fatalf("unbound profile = %v", got)
		}
		read := h.invoker.Readiness(ctx, "harbor.example", model.Ref{ID: "CAP-READ-ORDER-EXCEPTION"}, profile, "order-read")
		if len(read) != 1 || read[0].Requirement != "CHR-BIND-006" {
			t.Fatalf("capability without an authority binding = %+v", read)
		}
		if got := requirements(h.invoker.Readiness(ctx, "harbor.example", model.Ref{ID: "CAP-MISSING"}, profile)); len(got) != 1 || got[0] != "CHR-CAP-001" {
			t.Fatalf("unknown contract = %v", got)
		}
	})

	t.Run("schemas, shapes, predicates, and partial effects", func(t *testing.T) {
		h := newHarness(t)
		shaped := &model.DataSchema{InformationID: &model.Ref{ID: "INFO-ORDER-EXCEPTION"}}
		h.amend(func(c *model.CapabilityContract) {
			c.InputSchema = shaped
			c.Preconditions = append(c.Preconditions, model.Precondition{ID: "PRE", Statement: "s", Predicate: &model.Predicate{Language: model.PredicateCEL, Expression: "true"}})
			c.Postconditions = append(c.Postconditions, model.Postcondition{ID: "POST-2", Statement: "s", VerificationRequired: true, Outcomes: []string{"reserved"}})
		})
		got := requirements(h.invoker.Readiness(ctx, "harbor.example", reservation, profile, "reservation-create"))
		if want := []string{"CHR-CAP-011", "CHR-CAP-012", "CHR-EVID-016"}; !slices.Equal(got, want) {
			t.Fatalf("blockers = %v, want %v", got, want)
		}
		h.invoker.Definitions, h.invoker.Predicates = information.NewBaseProvider(h.corpus), &predicate.CEL{}
		h.invoker.Escalation = &capability.BaseEscalator{}
		if got := requirements(h.invoker.Readiness(ctx, "harbor.example", reservation, profile, "reservation-create")); len(got) != 0 {
			t.Fatalf("composed = %v", got)
		}
		h.amend(func(c *model.CapabilityContract) {
			c.InputSchema = &model.DataSchema{InformationID: &model.Ref{ID: "INFO-CREDIT-STATUS-SUMMARY"}}
		})
		if got := h.invoker.Readiness(ctx, "harbor.example", reservation, profile, "reservation-create"); len(got) != 0 {
			t.Fatalf("a shape without a data binding on the profile = %+v", got)
		}
		d, err := (corpus.Parser{}).Parse([]byte(`{"charterSpecVersion":"1.6.0","namespace":"harbor.example","kind":"DataBinding","id":"DATABIND-CREDIT","bindingVersion":"1",
			"systemProfileId":"SYSPROFILE-HARBOR-S4-2602","informationDefinitionId":"INFO-CREDIT-STATUS-SUMMARY","featureSupport":[{"feature":"credit-read","support":"supported"}],
			"fieldMappings":[{"charterPath":"customerId","externalPath":"Customer"},{"charterPath":"rating","externalPath":"Rating"}]}`))
		if err != nil {
			t.Fatal(err)
		}
		if err := h.corpus.Add(d); err != nil {
			t.Fatal(err)
		}
		unmapped := h.invoker.Readiness(ctx, "harbor.example", reservation, profile, "reservation-create")
		if got := requirements(unmapped); !slices.Equal(got, []string{"CHR-INFO-010", "CHR-INFO-010"}) {
			t.Fatalf("unresolved path and unmapped required property = %+v", unmapped)
		}
	})

	if got := requirements((&capability.BaseInvoker{}).Readiness(ctx, "harbor.example", reservation, profile)); len(got) != 1 || got[0] != "CHR-AUTH-010" {
		t.Fatalf("zero invoker = %v", got)
	}
}
