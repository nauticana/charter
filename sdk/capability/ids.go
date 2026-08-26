package capability

import (
	"fmt"
	"strings"
	"sync"

	"github.com/nauticana/charter/sdk/model"
)

// IDGenerator mints identifiers for the evidence an invoker appends.
type IDGenerator interface {
	NewID(kind model.Kind) string
}

// BaseCounterIDs mints sequential identifiers for one process; distributed runtimes supply their own generator.
type BaseCounterIDs struct {
	Prefix string
	mu     sync.Mutex
	n      int
}

var _ IDGenerator = (*BaseCounterIDs)(nil)

func (g *BaseCounterIDs) NewID(kind model.Kind) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.n++
	return fmt.Sprintf("%s%s-%d", g.Prefix, Prefix(kind), g.n)
}

// Prefix is the identifier prefix conventionally used for evidence of a kind.
func Prefix(kind model.Kind) string {
	if kind == model.KindActionRecord {
		return "ACT"
	}
	return strings.ToUpper(string(kind))
}
