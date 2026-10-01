package keel

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nauticana/keel/common"
	kmodel "github.com/nauticana/keel/model"

	"github.com/nauticana/charter/sdk/identity"
	"github.com/nauticana/charter/sdk/model"
)

var membershipBegda = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

func newHumanAccounts(t *testing.T) (*HumanAccounts, *DocumentStore, *memoryDB) {
	t.Helper()
	store, db := newDocumentStore(t)
	for _, user := range []int64{42, 43} {
		db.state.memberships[membershipKey{partner, user, membershipBegda}] = nil
	}
	accounts, err := NewHumanAccounts(context.Background(), db, func() time.Time { return at })
	if err != nil {
		t.Fatal(err)
	}
	return accounts, store, db
}

func alexLink() HumanAccount {
	return HumanAccount{PartnerID: partner, Human: model.Ref{Namespace: ns, ID: "HUMAN-ALEX-RIVERA"}, UserID: 42, PartnerUserBegda: membershipBegda,
		Begda: membershipBegda.Add(time.Hour)}
}

func TestHumanAccountLinksNeverOverlap(t *testing.T) {
	ctx := context.Background()
	accounts, _, db := newHumanAccounts(t)
	if err := accounts.Link(ctx, alexLink()); err != nil {
		t.Fatal(err)
	}
	if err := accounts.Link(ctx, alexLink()); err != nil || len(db.state.accounts) != 1 {
		t.Fatalf("an identical replay is not a no-op: %v, %d rows", err, len(db.state.accounts))
	}
	otherHuman := alexLink()
	otherHuman.Human.ID = "HUMAN-ELENA-TORRES"
	if err := accounts.Link(ctx, otherHuman); !errors.Is(err, ErrAccountLinkOverlap) {
		t.Errorf("a user acting as two humans: %v", err)
	}
	otherUser := alexLink()
	otherUser.UserID = 43
	if err := accounts.Link(ctx, otherUser); !errors.Is(err, ErrAccountLinkOverlap) {
		t.Errorf("a human acting through two users: %v", err)
	}
	for name, tc := range map[string]struct {
		edit func(*HumanAccount)
		want error
	}{
		"not a human":           {func(l *HumanAccount) { l.Human.ID = "POS-CREDIT-MANAGER" }, ErrNotHumanIdentity},
		"unknown document":      {func(l *HumanAccount) { l.Human.ID = "HUMAN-NONE" }, ErrNotHumanIdentity},
		"no membership":         {func(l *HumanAccount) { l.UserID = 44 }, ErrNoMembership},
		"before membership":     {func(l *HumanAccount) { l.Begda = membershipBegda.Add(-time.Hour) }, ErrInvalidAccountLink},
		"reversed period":       {func(l *HumanAccount) { e := l.Begda; l.Endda = &e }, ErrInvalidAccountLink},
		"no partner":            {func(l *HumanAccount) { l.PartnerID = 0 }, ErrInvalidAccountLink},
		"no membership begda":   {func(l *HumanAccount) { l.PartnerUserBegda = time.Time{} }, ErrInvalidAccountLink},
		"membership of another": {func(l *HumanAccount) { l.PartnerUserBegda = membershipBegda.Add(time.Minute) }, ErrNoMembership},
	} {
		link := alexLink()
		link.Human.ID, link.UserID = "HUMAN-JORDAN-KIM", 43
		tc.edit(&link)
		if err := accounts.Link(ctx, link); !errors.Is(err, tc.want) {
			t.Errorf("%s: %v, want %v", name, err, tc.want)
		}
	}
	ended := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	db.state.memberships[membershipKey{partner, 43, membershipBegda}] = &ended
	outlives := alexLink()
	outlives.Human.ID, outlives.UserID = "HUMAN-JORDAN-KIM", 43
	if err := accounts.Link(ctx, outlives); !errors.Is(err, ErrNoMembership) {
		t.Errorf("a link outliving its membership: %v", err)
	}
	if len(db.state.accounts) != 1 {
		t.Fatalf("refused links were stored: %d rows", len(db.state.accounts))
	}
	db.state.memberships[membershipKey{partner, 43, membershipBegda}] = nil
	future := alexLink()
	future.Human.ID, future.UserID = "HUMAN-JORDAN-KIM", 43
	future.Begda = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	end := future.Begda.Add(24 * time.Hour)
	future.Endda = &end
	if err := accounts.Link(ctx, future); err != nil {
		t.Fatal(err)
	}
	overlap := future
	overlap.Begda, overlap.Endda = future.Begda.Add(time.Hour), nil
	if err := accounts.Link(ctx, overlap); !errors.Is(err, ErrAccountLinkOverlap) {
		t.Errorf("an open link did not overlap a finite link beyond year 9999: %v", err)
	}
}

func TestHumanAccountsResolveTheLinkEffectiveAtATime(t *testing.T) {
	ctx := context.Background()
	accounts, _, db := newHumanAccounts(t)
	alex := alexLink()
	if err := accounts.Link(ctx, alex); err != nil {
		t.Fatal(err)
	}
	if human, err := accounts.Human(ctx, partner, 42, at); err != nil || human != alex.Human {
		t.Fatalf("human of user = %+v %v", human, err)
	}
	if user, err := accounts.User(ctx, partner, alex.Human, at); err != nil || user != 42 {
		t.Fatalf("user of human = %d %v", user, err)
	}
	if _, err := accounts.Human(ctx, partner, 42, membershipBegda); !errors.Is(err, ErrUnmapped) {
		t.Errorf("resolved before the link began: %v", err)
	}
	if _, err := accounts.Human(ctx, partner+1, 42, at); !errors.Is(err, ErrUnmapped) {
		t.Errorf("resolved in another tenant: %v", err)
	}

	end := at.Add(-time.Hour)
	if err := accounts.End(ctx, partner, alex.Human, end); err != nil {
		t.Fatal(err)
	}
	if _, err := accounts.Human(ctx, partner, 42, at); !errors.Is(err, ErrUnmapped) {
		t.Errorf("resolved after the link ended: %v", err)
	}
	if human, err := accounts.Human(ctx, partner, 42, end.Add(-time.Minute)); err != nil || human != alex.Human {
		t.Errorf("history before the end: %+v %v", human, err)
	}
	if err := accounts.End(ctx, partner, alex.Human, at); !errors.Is(err, ErrUnmapped) {
		t.Errorf("ended a link that is not open: %v", err)
	}
	successor := alexLink()
	successor.Human.ID, successor.Begda = "HUMAN-ELENA-TORRES", end
	if err := accounts.Link(ctx, successor); err != nil {
		t.Fatalf("a link following an ended one: %v", err)
	}
	membershipEnd := at.Add(-time.Minute)
	db.state.memberships[membershipKey{partner, 42, membershipBegda}] = &membershipEnd
	if _, err := accounts.Human(ctx, partner, 42, at); !errors.Is(err, ErrUnmapped) {
		t.Errorf("resolved after the membership ended: %v", err)
	}
	if _, err := NewHumanAccounts(ctx, nil, nil); !errors.Is(err, ErrNoDatabase) {
		t.Errorf("accounts without a database: %v", err)
	}
}

func userContext(subject string, claims map[string]any) context.Context {
	p := &kmodel.TokenPrincipal{Subject: subject, Claims: claims}
	ctx := context.WithValue(context.Background(), common.AuthPrincipal, p)
	return context.WithValue(ctx, common.PartnerID, partner)
}

func TestTableIdentityMapResolvesHumansThroughTheirAccountLink(t *testing.T) {
	ctx := context.Background()
	accounts, store, _ := newHumanAccounts(t)
	if err := accounts.Link(ctx, alexLink()); err != nil {
		t.Fatal(err)
	}
	m := TableIdentityMap{Accounts: accounts, Agents: BaseClaimIdentityMap{Namespace: ns}, Now: func() time.Time { return at }}
	alex := model.ObjectRef{Kind: model.KindHumanIdentity, Namespace: ns, ID: "HUMAN-ALEX-RIVERA"}

	history := []any{map[string]any{"state": "active", "effectiveAt": "2025-01-01T00:00:00Z"}}
	if err := store.Save(ctx, partner, edited(t, store, alex.ID, "lifecycleHistory", history)); err != nil {
		t.Fatal(err)
	}
	caller := Caller{Identities: identity.NewBaseResolver(store.Source(partner)), Map: m}
	actor, _, err := caller.Actor(userContext("user:42", nil), at)
	if err != nil || actor.ID != alex.ID || actor.Kind != model.KindHumanIdentity {
		t.Fatalf("actor of user:42 = %+v %v", actor, err)
	}
	if principal, err := m.Principal(userContext("user:42", nil), alex); err != nil || principal.Kind != kmodel.PrincipalUser || principal.ID != 42 {
		t.Fatalf("principal = %+v %v", principal, err)
	}
	relativeAlex := alex
	relativeAlex.Namespace = ""
	if _, err := m.Principal(userContext("user:42", nil), relativeAlex); err != nil {
		t.Errorf("relative actor in its mapped namespace: %v", err)
	}
	claimed := userContext("external-subject", map[string]any{DefaultUserIDClaim: float64(42)})
	if ref, err := m.Actor(claimed, mustSession(t, claimed)); err != nil || ref != alex {
		t.Errorf("user id claim: %+v %v", ref, err)
	}
	forged := userContext("user:42", map[string]any{DefaultKindClaim: string(model.KindHumanIdentity), DefaultIDClaim: "HUMAN-ELENA-TORRES"})
	if ref, err := m.Actor(forged, mustSession(t, forged)); err != nil || ref != alex {
		t.Errorf("a human identity claim overrode the account link: %+v %v", ref, err)
	}
	for name, ctx := range map[string]context.Context{
		"unlinked user":   userContext("user:43", nil),
		"foreign subject": userContext("someone", nil),
		"bad claim":       userContext("user:42", map[string]any{DefaultUserIDClaim: "x"}),
	} {
		if _, err := m.Actor(ctx, mustSession(t, ctx)); !errors.Is(err, ErrUnmapped) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := m.Principal(userContext("user:42", nil), model.ObjectRef{Kind: model.KindHumanIdentity, Namespace: ns, ID: "HUMAN-ELENA-TORRES"}); !errors.Is(err, ErrUnmapped) {
		t.Errorf("principal for an actor the session is not: %v", err)
	}

	agentSession := userContext("agent-sub", agentClaims())
	if principal, err := m.Principal(agentSession, model.ObjectRef{Kind: model.KindAgentIdentity, ID: agent}); err != nil || principal.Kind != AgentPrincipalKind || principal.ID != agent {
		t.Errorf("agent principal: %+v %v", principal, err)
	}
	if _, err := (TableIdentityMap{}).Actor(ctx, mustSession(t, userContext("user:42", nil))); !errors.Is(err, ErrUnmapped) {
		t.Errorf("map without accounts: %v", err)
	}
}

func mustSession(t *testing.T, ctx context.Context) common.CallerSession {
	t.Helper()
	s, err := common.CallerSessionFromContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
