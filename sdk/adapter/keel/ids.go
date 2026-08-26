package keel

import (
	"fmt"

	"github.com/nauticana/keel/port"

	"github.com/nauticana/charter/sdk/capability"
	"github.com/nauticana/charter/sdk/model"
)

// BigintIDs mints Charter evidence identifiers from keel's multi-node bigint generator.
type BigintIDs struct {
	Generator port.BigintGenerator
	Prefix    string
}

var _ capability.IDGenerator = (*BigintIDs)(nil)

func (g *BigintIDs) NewID(kind model.Kind) string {
	return fmt.Sprintf("%s%s-%d", g.Prefix, capability.Prefix(kind), g.Generator.NextID())
}
