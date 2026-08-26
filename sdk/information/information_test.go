package information

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func TestHarborGovernance(t *testing.T) {
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	e := &BaseEvaluator{Provider: NewBaseProvider(c)}
	at := time.Date(2026, 6, 18, 17, 0, 0, 0, time.UTC)
	info := model.Ref{ID: "INFO-CREDIT-STATUS-SUMMARY"}
	if d := e.Use(ctx, UseRequest{Namespace: "harbor.example", Information: info, Purpose: "audit", At: at}); d.Result != Allowed || len(d.Policies) != 1 {
		t.Errorf("audit: %+v", d)
	}
	if d := e.Use(ctx, UseRequest{Namespace: "harbor.example", Information: info, Purpose: "marketing analysis", At: at}); d.Result != Denied || d.Requirement != "CHR-INFO-007" {
		t.Errorf("undeclared purpose: %+v", d)
	}
	if d := e.Use(ctx, UseRequest{Namespace: "harbor.example", Information: model.Ref{ID: "INFO-NOWHERE"}, Purpose: "audit", At: at}); d.Result != Error {
		t.Errorf("unknown information: %+v", d)
	}
	if r, err := e.Retention(ctx, "harbor.example", info, model.ArtifactPrompt, at); err != nil || r.RetentionRule != "90 days" {
		t.Errorf("prompt retention: %+v %v", r, err)
	}
	if _, err := e.Retention(ctx, "harbor.example", info, model.ArtifactSourceData, at); !errors.Is(err, ErrNoRetention) {
		t.Errorf("undeclared artifact kind: %v", err)
	}
	if cs, err := e.AccessConstraints(ctx, "harbor.example", info, at); err != nil || len(cs) != 2 {
		t.Errorf("access constraints: %v %v", cs, err)
	}
}

func policies(t *testing.T, behavior string, precedenceA, precedenceB int) *BaseEvaluator {
	policy := func(id string, precedence int, uses string) string {
		return fmt.Sprintf(`{"charterSpecVersion":"draft","namespace":"t","kind":"InformationGovernancePolicy","id":"%s","name":"%s","scope":[{"kind":"InformationDefinition","id":"I"}],"permittedUses":[%s],"accessConstraints":["c"],"retentionAndDeletion":[{"artifactKind":"prompt","retentionRule":"%s","deletionRule":"purge"}],"precedence":%d,"conflictBehavior":"%s"}`, id, id, uses, id, precedence, behavior)
	}
	docs := []string{
		`{"charterSpecVersion":"draft","namespace":"t","kind":"InformationDefinition","id":"I","name":"i","businessMeaning":"m","ownerOrSteward":{"kind":"OrganizationUnit","id":"OU"},"classification":"internal","governancePolicyIds":["A","B"]}`,
		policy("A", precedenceA, `"audit","export"`),
		policy("B", precedenceB, `"audit"`),
	}
	c := corpus.New()
	for _, raw := range docs {
		d, err := (corpus.Parser{}).Parse([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if err := c.Add(d); err != nil {
			t.Fatal(err)
		}
	}
	return &BaseEvaluator{Provider: NewBaseProvider(c)}
}

func TestConflictsFailClosedOrFollowPrecedence(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	req := UseRequest{Namespace: "t", Information: model.Ref{ID: "I"}, Purpose: "export", At: at}
	if d := policies(t, model.ConflictFailClosed, 10, 1).Use(ctx, req); d.Result != Denied || d.Requirement != "CHR-INFO-008" {
		t.Errorf("fail-closed conflict: %+v", d)
	}
	if d := policies(t, model.ConflictPrecedence, 10, 1).Use(ctx, req); d.Result != Allowed {
		t.Errorf("permitting policy with precedence: %+v", d)
	}
	if d := policies(t, model.ConflictPrecedence, 1, 10).Use(ctx, req); d.Result != Denied {
		t.Errorf("denying policy with precedence: %+v", d)
	}
	if d := policies(t, model.ConflictPrecedence, 5, 5).Use(ctx, req); d.Result != Denied {
		t.Errorf("equal precedence must deny: %+v", d)
	}
	if r, err := policies(t, model.ConflictPrecedence, 1, 10).Retention(ctx, "t", model.Ref{ID: "I"}, model.ArtifactPrompt, at); err != nil || r.RetentionRule != "B" {
		t.Errorf("retention precedence: %+v %v", r, err)
	}
	if _, err := policies(t, model.ConflictFailClosed, 1, 10).Retention(ctx, "t", model.Ref{ID: "I"}, model.ArtifactPrompt, at); !errors.Is(err, ErrConflict) {
		t.Errorf("retention conflict must fail closed: %v", err)
	}
}
