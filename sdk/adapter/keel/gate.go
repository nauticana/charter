package keel

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nauticana/keel/common"

	"github.com/nauticana/charter/sdk/authority"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/identity"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/temporal"
)

var (
	ErrUnmappedRoute = errors.New("request maps to no Charter capability")
	ErrNoAuthority   = errors.New("no effective grant for the capability")
)

// Gated is the Charter capability a keel boundary protects; empty scope and context match any grant.
type Gated struct {
	Namespace             string
	EnterpriseID          model.Ref
	CapabilityID          model.Ref
	ResourceScope         string
	OrganizationalContext string
}

// Routes maps an HTTP request to the capability it exercises; unmapped requests are refused.
type Routes interface {
	Capability(r *http.Request) (Gated, bool)
}

// BaseRouteTable matches "METHOD /path" exactly, then the longest "METHOD /prefix/" entry.
type BaseRouteTable map[string]Gated

var _ Routes = BaseRouteTable{}

func (t BaseRouteTable) Capability(r *http.Request) (Gated, bool) {
	if g, ok := t[r.Method+" "+r.URL.Path]; ok {
		return g, true
	}
	var best string
	var found Gated
	for key, g := range t {
		method, prefix, ok := strings.Cut(key, " ")
		if ok && method == r.Method && strings.HasSuffix(prefix, "/") && strings.HasPrefix(r.URL.Path, prefix) && len(prefix) > len(best) {
			best, found = prefix, g
		}
	}
	return found, best != ""
}

// Clearance is what a boundary established: the active actor, its session, and the grant that authorizes the capability.
type Clearance struct {
	Actor      identity.Actor
	Session    common.CallerSession
	Capability Gated
	GrantRef   model.Ref
}

type clearanceKey struct{}

func ClearanceFromContext(ctx context.Context) (Clearance, bool) {
	c, ok := ctx.Value(clearanceKey{}).(Clearance)
	return c, ok
}

// Gate establishes the acting identity and the presence of effective authority for a capability at keel's HTTP,
// table-action, and worker boundaries (CHR-SEC-001, CHR-AUTH-002). Limits, approvals, separation of duties, and
// information governance stay with the invoker at the action boundary (CHR-SEC-003).
type Gate struct {
	Caller Caller
	Grants authority.GrantSource
	Now    func() time.Time
}

func (g *Gate) now() time.Time {
	if g.Now != nil {
		return g.Now()
	}
	return time.Now().UTC()
}

func (g *Gate) Check(ctx context.Context, cap Gated) (Clearance, error) {
	if g.Grants == nil {
		return Clearance{}, errors.New("gate has no grant source")
	}
	at := g.now()
	actor, session, err := g.Caller.Actor(ctx, at)
	if err != nil {
		return Clearance{}, err
	}
	grants, err := g.Grants.Grants(ctx, authority.Request{Namespace: cap.Namespace, EnterpriseID: cap.EnterpriseID, Actor: actor.Ref(),
		CapabilityID: cap.CapabilityID, ResourceScope: cap.ResourceScope, OrganizationalContext: cap.OrganizationalContext, At: at})
	if err != nil {
		return Clearance{}, err
	}
	for _, grant := range grants {
		if grant.Actor.Kind == actor.Kind && corpus.ObjectKeyOf(grant.Namespace, grant.Actor) == corpus.ObjectKeyOf(cap.Namespace, actor.Ref()) &&
			corpus.KeyOf(grant.Namespace, grant.CapabilityID) == corpus.KeyOf(cap.Namespace, cap.CapabilityID) &&
			grant.EnterpriseID != nil && corpus.KeyOf(grant.Namespace, *grant.EnterpriseID) == corpus.KeyOf(cap.Namespace, cap.EnterpriseID) &&
			grant.Validity != nil && temporal.EffectiveAt(grant.Validity, at) &&
			(cap.ResourceScope == "" || grant.ResourceScope == cap.ResourceScope) &&
			(cap.OrganizationalContext == "" || grant.OrganizationalContext == cap.OrganizationalContext) {
			return Clearance{Actor: actor, Session: session, Capability: cap, GrantRef: model.Ref{Namespace: grant.Namespace, ID: grant.ID}}, nil
		}
	}
	return Clearance{}, fmt.Errorf("%w: %s %s on %s", ErrNoAuthority, actor.Kind, actor.ID, cap.CapabilityID.ID)
}

// Middleware refuses requests that map to no capability or whose caller holds no effective grant for it, and hands
// the clearance to the next handler through the context.
func (g *Gate) Middleware(routes Routes) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if routes == nil || next == nil {
				common.WriteJSONError(w, http.StatusInternalServerError, "authority gate is not fully composed")
				return
			}
			cap, ok := routes.Capability(r)
			if !ok {
				common.WriteJSONError(w, http.StatusForbidden, ErrUnmappedRoute.Error())
				return
			}
			g.serve(w, r, cap, next.ServeHTTP)
		})
	}
}

// Handler gates one capability for a table action or custom endpoint; place it where keel's Routes map expects the
// handler, alone or inside handler.WrapTableAction.
func (g *Gate) Handler(cap Gated, inner http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { g.serve(w, r, cap, inner) }
}

func (g *Gate) serve(w http.ResponseWriter, r *http.Request, cap Gated, next http.HandlerFunc) {
	if next == nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "authority gate has no downstream handler")
		return
	}
	clearance, err := g.Check(r.Context(), cap)
	switch {
	case err == nil:
		next(w, r.WithContext(context.WithValue(r.Context(), clearanceKey{}, clearance)))
	case errors.Is(err, common.ErrUnauthenticated), errors.Is(err, common.ErrCallerIdentityConflict):
		common.WriteJSONError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, ErrNoAuthority), errors.Is(err, ErrUnmapped), errors.Is(err, corpus.ErrNotFound), errors.Is(err, identity.ErrNotIdentity),
		errors.Is(err, identity.ErrInactive), errors.Is(err, identity.ErrLifecycle):
		common.WriteJSONError(w, http.StatusForbidden, err.Error())
	default:
		common.WriteJSONError(w, http.StatusInternalServerError, "authority could not be established")
	}
}
