package information

import (
	"context"
	"time"

	"github.com/nauticana/charter/sdk/model"
)

type Result string

const (
	Allowed Result = "allowed"
	Denied  Result = "denied"
	Error   Result = "error"
)

// Decision is the fail-closed outcome of a governance question with the policies consulted and the requirement applied.
type Decision struct {
	Result      Result
	Reason      string
	Requirement string
	Policies    []model.Ref
}

// UseRequest asks whether information may be used for a declared purpose at a time (CHR-INFO-007).
type UseRequest struct {
	Namespace   string
	Information model.Ref
	Purpose     string
	At          time.Time
}

type Evaluator interface {
	Use(ctx context.Context, req UseRequest) Decision
	Retention(ctx context.Context, ownerNamespace string, information model.Ref, artifactKind string, at time.Time) (model.RetentionRule, error)
	AccessConstraints(ctx context.Context, ownerNamespace string, information model.Ref, at time.Time) ([]string, error)
}
