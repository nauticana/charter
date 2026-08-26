// Package information resolves information definitions and governance policies and decides permitted use, access
// constraints, and retention with declared precedence or fail-closed conflict handling.
package information

import (
	"context"

	"github.com/nauticana/charter/sdk/model"
)

type Provider interface {
	Definition(ctx context.Context, ownerNamespace string, ref model.Ref) (model.InformationDefinition, error)
	Policy(ctx context.Context, ownerNamespace string, ref model.Ref) (model.InformationGovernancePolicy, error)
}
