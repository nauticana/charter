// Package keel adapts Charter's neutral contracts to the primitives keel exposes: the authenticated caller session,
// RBAC permission checks, trust guards, the table change logger, the idempotency ledger, message publishing, metrics,
// and request correlation ids. Keel never depends on Charter; this package is the optional bridge.
package keel

import (
	"context"
	"errors"
	"regexp"

	"github.com/nauticana/keel/common"

	"github.com/nauticana/charter/sdk/model"
)

var ErrNoRequestID = errors.New("no usable request correlation id in context")

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)

// RuntimeContext attributes an action to the runtime instance and keel's request correlation id (CHR-AGENT-006).
func RuntimeContext(ctx context.Context, runtime model.Ref) (model.RuntimeContext, error) {
	id := common.RequestIDFromContext(ctx)
	if !identifier.MatchString(id) {
		return model.RuntimeContext{}, ErrNoRequestID
	}
	return model.RuntimeContext{RuntimeInstanceID: runtime, ExecutionContextID: id}, nil
}
