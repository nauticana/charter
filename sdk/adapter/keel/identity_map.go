package keel

import (
	"context"
	"errors"
	"fmt"

	"github.com/nauticana/keel/common"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

var ErrUnmapped = errors.New("keel identity does not map to a Charter identity")

// IdentityMap connects keel's authenticated subjects and numeric user ids to Charter identities.
// Credentials never define the identity: a rotated token or key maps to the same actor (CHR-ID-003, CHR-ID-007).
type IdentityMap interface {
	Actor(ctx context.Context, s Session) (model.ObjectRef, error)
	UserID(ctx context.Context, actor model.ObjectRef) (int, error)
}

const (
	DefaultKindClaim   = "charter_kind"
	DefaultIDClaim     = "charter_id"
	DefaultUserIDClaim = "keel_user_id"
)

// BaseClaimIdentityMap reads the Charter identity and keel user id from claims the enterprise's authorization server
// mints into access tokens; sessions without a principal are unmapped.
type BaseClaimIdentityMap struct {
	Namespace   string
	KindClaim   string
	IDClaim     string
	UserIDClaim string
}

var _ IdentityMap = BaseClaimIdentityMap{}

func (m BaseClaimIdentityMap) Actor(_ context.Context, s Session) (model.ObjectRef, error) {
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

// UserID returns the keel user id claimed by the session, provided the session acts as the given actor.
func (m BaseClaimIdentityMap) UserID(ctx context.Context, actor model.ObjectRef) (int, error) {
	s, err := SessionFromContext(ctx)
	if err != nil {
		return 0, err
	}
	mapped, err := m.Actor(ctx, s)
	if err != nil {
		return 0, err
	}
	if mapped.Kind != actor.Kind || corpus.ObjectKeyOf(m.Namespace, mapped) != corpus.ObjectKeyOf(m.Namespace, actor) {
		return 0, fmt.Errorf("%w: session acts as %s %s, not %s %s", ErrUnmapped, mapped.Kind, mapped.ID, actor.Kind, actor.ID)
	}
	userID, ok := common.AsInt64OK(s.Principal.Claims[claim(m.UserIDClaim, DefaultUserIDClaim)])
	if !ok || userID <= 0 {
		return 0, fmt.Errorf("%w: no keel user id claim for subject %q", ErrUnmapped, s.Principal.Subject)
	}
	return int(userID), nil
}

func claim(name, fallback string) string {
	if name == "" {
		return fallback
	}
	return name
}
