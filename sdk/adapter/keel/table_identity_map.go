package keel

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nauticana/keel/common"
	kmodel "github.com/nauticana/keel/model"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// KeelUserSubjectPrefix is the subject prefix keel's authorization server gives a user's access token.
const KeelUserSubjectPrefix = "user:"

// TableIdentityMap resolves a human session through charter_human_account instead of Charter identity claims: the
// session's keel user in its partner maps to the HumanIdentity linked at the current time. Agent sessions keep the
// claim mapping of Agents, so their keel principal remains Principal{Kind: AgentPrincipalKind}.
type TableIdentityMap struct {
	Accounts *HumanAccounts
	Agents   BaseClaimIdentityMap
	// Now is the clock links are resolved at; nil uses the system clock.
	Now func() time.Time
}

var _ IdentityMap = TableIdentityMap{}

func (m TableIdentityMap) Actor(ctx context.Context, s common.CallerSession) (model.ObjectRef, error) {
	if m.agentSession(s) {
		return m.Agents.Actor(ctx, s)
	}
	if m.Accounts == nil {
		return model.ObjectRef{}, fmt.Errorf("%w: no human account store", ErrUnmapped)
	}
	userID, err := m.userID(s)
	if err != nil {
		return model.ObjectRef{}, err
	}
	human, err := m.Accounts.Human(ctx, s.PartnerID, userID, m.now())
	if err != nil {
		return model.ObjectRef{}, err
	}
	return model.ObjectRef{Kind: model.KindHumanIdentity, Namespace: human.Namespace, ID: human.ID}, nil
}

// Principal returns the keel principal of the actor the session acts as; a human maps to the session's own user.
func (m TableIdentityMap) Principal(ctx context.Context, actor model.ObjectRef) (kmodel.Principal, error) {
	s, err := common.CallerSessionFromContext(ctx)
	if err != nil {
		return kmodel.Principal{}, err
	}
	if m.agentSession(s) {
		return m.Agents.Principal(ctx, actor)
	}
	mapped, err := m.Actor(ctx, s)
	if err != nil {
		return kmodel.Principal{}, err
	}
	if actor.Kind != model.KindHumanIdentity || corpus.ObjectKeyOf("", mapped) != corpus.ObjectKeyOf(mapped.Namespace, actor) {
		return kmodel.Principal{}, fmt.Errorf("%w: session acts as %s %s, not %s %s", ErrUnmapped, mapped.Kind, mapped.ID, actor.Kind, actor.ID)
	}
	userID, err := m.userID(s)
	if err != nil {
		return kmodel.Principal{}, err
	}
	return kmodel.UserPrincipal(int(userID)), nil
}

func (m TableIdentityMap) agentSession(s common.CallerSession) bool {
	if s.Principal == nil {
		return false
	}
	kind, _ := s.Principal.Claims[claim(m.Agents.KindClaim, DefaultKindClaim)].(string)
	return kind == string(model.KindAgentIdentity)
}

// userID reads the keel user id claim, or else the user subject keel's authorization server mints.
func (m TableIdentityMap) userID(s common.CallerSession) (int64, error) {
	if s.Principal == nil {
		return 0, fmt.Errorf("%w: subject %q carries no token", ErrUnmapped, s.Subject)
	}
	if v, present := s.Principal.Claims[claim(m.Agents.UserIDClaim, DefaultUserIDClaim)]; present {
		if id, ok := common.AsInt64OK(v); ok && id > 0 {
			return id, nil
		}
		return 0, fmt.Errorf("%w: invalid keel user id claim for subject %q", ErrUnmapped, s.Principal.Subject)
	}
	if digits, ok := strings.CutPrefix(s.Principal.Subject, KeelUserSubjectPrefix); ok {
		if id, err := strconv.ParseInt(digits, 10, 64); err == nil && id > 0 {
			return id, nil
		}
	}
	return 0, fmt.Errorf("%w: subject %q names no keel user", ErrUnmapped, s.Principal.Subject)
}

func (m TableIdentityMap) now() time.Time {
	if m.Now == nil {
		return time.Now()
	}
	return m.Now()
}
