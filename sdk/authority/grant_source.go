package authority

import (
	"context"

	"github.com/nauticana/charter/sdk/model"
)

type GrantSource interface {
	Grants(ctx context.Context, req Request) ([]model.AuthorityGrant, error)
}
