package keel

import (
	"context"
	"fmt"

	"github.com/nauticana/charter/sdk/authority"
	"github.com/nauticana/charter/sdk/corpus"
)

// PermissionChecker is keel's RBAC hook; port.DatabaseRepository satisfies it through CheckActionPermission.
type PermissionChecker interface {
	CheckActionPermission(ctx context.Context, userID int, authObject, action, scope string) (allowed bool, ownScope bool)
}

// Permission names the keel authorization object, action, and scope that enforce one capability.
type Permission struct {
	AuthObject string
	Action     string
	Scope      string
}

// PermissionGate layers keel RBAC behind a Charter evaluator: keel may deny an action Charter allows but never allows one
// Charter denies, and a capability with no mapped permission fails closed (CHR-ENT-007, CHR-SEC-003, CHR-BIND-006).
type PermissionGate struct {
	Charter     authority.Evaluator
	Keel        PermissionChecker
	Identities  IdentityMap
	Permissions map[corpus.DocumentKey]Permission
}

var _ authority.Evaluator = (*PermissionGate)(nil)

func (g *PermissionGate) Evaluate(ctx context.Context, req authority.Request) authority.Decision {
	if g.Charter == nil || g.Keel == nil || g.Identities == nil {
		return authority.Decision{Result: authority.Error, Reason: "permission gate is not fully composed"}
	}
	d := g.Charter.Evaluate(ctx, req)
	if d.Result != authority.Allowed {
		return d
	}
	perm, ok := g.Permissions[corpus.KeyOf(req.Namespace, req.CapabilityID)]
	if !ok {
		return authority.Decision{GrantRef: d.GrantRef, Result: authority.Denied, Reason: fmt.Sprintf("no keel permission is mapped for capability %s", req.CapabilityID.ID)}
	}
	userID, err := g.Identities.UserID(ctx, req.Actor)
	if err != nil {
		return authority.Decision{GrantRef: d.GrantRef, Result: authority.Error, Reason: err.Error()}
	}
	if allowed, _ := g.Keel.CheckActionPermission(ctx, userID, perm.AuthObject, perm.Action, perm.Scope); !allowed {
		return authority.Decision{GrantRef: d.GrantRef, Result: authority.Denied, Reason: fmt.Sprintf("keel denies %s/%s on %q for user %d", perm.AuthObject, perm.Action, perm.Scope, userID)}
	}
	d.Reason += fmt.Sprintf("; keel permission %s/%s confirmed", perm.AuthObject, perm.Action)
	return d
}
