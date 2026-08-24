package authority

import (
	"context"

	"github.com/nauticana/charter/sdk/model"
)

// BaseGrantSource serves grants from memory.
type BaseGrantSource struct {
	Items []model.AuthorityGrant
}

var _ GrantSource = (*BaseGrantSource)(nil)

func (s *BaseGrantSource) Grants(_ context.Context, req Request) ([]model.AuthorityGrant, error) {
	var out []model.AuthorityGrant
	for _, g := range s.Items {
		if sameObjectRef(g.Namespace, g.Actor, req.Namespace, req.Actor) &&
			sameRef(g.Namespace, g.CapabilityID, req.Namespace, req.CapabilityID) &&
			g.EnterpriseID != nil && sameRef(g.Namespace, *g.EnterpriseID, req.Namespace, req.EnterpriseID) {
			out = append(out, g)
		}
	}
	return out, nil
}
