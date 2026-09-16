package keel

import (
	"context"
	"errors"
	"fmt"

	"github.com/nauticana/keel/common"
	kmodel "github.com/nauticana/keel/model"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

var ErrUnmapped = errors.New("keel identity does not map to a Charter identity")

// IdentityMap connects keel's authenticated subjects to Charter identities and maps those actors back to the
// principal kind and id whose keel grants must be evaluated.
// Credentials never define the identity: a rotated token or key maps to the same actor (CHR-ID-003, CHR-ID-007).
type IdentityMap interface {
	Actor(ctx context.Context, s common.CallerSession) (model.ObjectRef, error)
	Principal(ctx context.Context, actor model.ObjectRef) (kmodel.Principal, error)
}

const (
	DefaultKindClaim   = "charter_kind"
	DefaultIDClaim     = "charter_id"
	DefaultUserIDClaim = "keel_user_id"
)

// AgentPrincipalKind is the keel grant kind for a Charter AgentIdentity: subject agent_id, sole filter tenant_id.
const AgentPrincipalKind kmodel.PrincipalKind = "agent"

// BaseClaimIdentityMap reads the Charter identity and, for humans, the keel user id from claims the enterprise's
// authorization server mints into access tokens; sessions without a token principal are unmapped.
type BaseClaimIdentityMap struct {
	Namespace   string
	KindClaim   string
	IDClaim     string
	UserIDClaim string
}

var _ IdentityMap = BaseClaimIdentityMap{}

func (m BaseClaimIdentityMap) Actor(_ context.Context, s common.CallerSession) (model.ObjectRef, error) {
	if s.Principal == nil {
		return model.ObjectRef{}, fmt.Errorf("%w: subject %q carries no claims", ErrUnmapped, s.Subject)
	}
	kind, _ := s.Principal.Claims[claim(m.KindClaim, DefaultKindClaim)].(string)
	id, _ := s.Principal.Claims[claim(m.IDClaim, DefaultIDClaim)].(string)
	if (kind != string(model.KindHumanIdentity) && kind != string(model.KindAgentIdentity)) || id == "" {
		return model.ObjectRef{}, fmt.Errorf("%w: subject %q", ErrUnmapped, s.Principal.Subject)
	}
	return model.ObjectRef{Kind: model.Kind(kind), ID: id, Namespace: m.Namespace}, nil
}

// Principal returns the keel grant principal for the actor established by the session. Human identities use the
// claimed keel user id. Agent identities use their stable Charter id and authenticated tenant directly.
func (m BaseClaimIdentityMap) Principal(ctx context.Context, actor model.ObjectRef) (kmodel.Principal, error) {
	s, err := common.CallerSessionFromContext(ctx)
	if err != nil {
		return kmodel.Principal{}, err
	}
	mapped, err := m.Actor(ctx, s)
	if err != nil {
		return kmodel.Principal{}, err
	}
	if mapped.Kind != actor.Kind || corpus.ObjectKeyOf(m.Namespace, mapped) != corpus.ObjectKeyOf(m.Namespace, actor) {
		return kmodel.Principal{}, fmt.Errorf("%w: session acts as %s %s, not %s %s", ErrUnmapped, mapped.Kind, mapped.ID, actor.Kind, actor.ID)
	}
	if actor.Kind == model.KindAgentIdentity {
		if s.PartnerID <= 0 {
			return kmodel.Principal{}, fmt.Errorf("%w: no keel tenant for agent %q", ErrUnmapped, actor.ID)
		}
		return kmodel.Principal{Kind: AgentPrincipalKind, ID: actor.ID, Scope: []any{s.PartnerID}}, nil
	}
	userID, ok := common.AsInt64OK(s.Principal.Claims[claim(m.UserIDClaim, DefaultUserIDClaim)])
	if !ok || userID <= 0 {
		return kmodel.Principal{}, fmt.Errorf("%w: no keel user id claim for subject %q", ErrUnmapped, s.Principal.Subject)
	}
	return kmodel.UserPrincipal(int(userID)), nil
}

func claim(name, fallback string) string {
	if name == "" {
		return fallback
	}
	return name
}
