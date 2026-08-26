// Package keel adapts Charter's neutral contracts to the primitives keel exposes: the authenticated principal and
// tenant context keys, RBAC permission checks, trust guards, the table change logger, message publishing, metrics,
// and request correlation ids. Keel never depends on Charter; this package is the optional bridge.
package keel

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/nauticana/keel/common"
	"github.com/nauticana/keel/port"

	"github.com/nauticana/charter/sdk/model"
)

var (
	ErrUnauthenticated = errors.New("no authenticated principal, subject, or api key in context")
	ErrNoRequestID     = errors.New("no usable request correlation id in context")
)

// Session is what keel's middleware established for the request: OAuth principal, subject, tenant, credential, scopes, and correlation id.
type Session struct {
	Principal *port.Principal
	Subject   string
	PartnerID int64
	APIKeyID  int64
	Scopes    []string
	RequestID string
}

// SessionFromContext reads the values keel's OAuth and API-key middlewares bind; an unauthenticated context fails closed (CHR-SEC-001).
func SessionFromContext(ctx context.Context) (Session, error) {
	s := Session{RequestID: common.RequestIDFromContext(ctx)}
	s.Principal, _ = ctx.Value(common.AuthPrincipal).(*port.Principal)
	s.Subject, _ = ctx.Value(common.Subject).(string)
	s.PartnerID, _ = ctx.Value(common.PartnerID).(int64)
	s.APIKeyID, _ = ctx.Value(common.ApiKeyID).(int64)
	if raw, ok := ctx.Value(common.Scopes).(string); ok {
		s.Scopes = strings.FieldsFunc(raw, func(r rune) bool { return r == ' ' || r == ',' })
	}
	if s.Principal != nil && s.Subject == "" {
		s.Subject = s.Principal.Subject
	}
	if s.Principal == nil && s.Subject == "" && s.APIKeyID == 0 {
		return Session{}, ErrUnauthenticated
	}
	return s, nil
}

func (s Session) HasScope(scope string) bool {
	for _, granted := range s.Scopes {
		if granted == scope {
			return true
		}
	}
	return false
}

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)

// RuntimeContext attributes an action to the runtime instance and keel's request correlation id (CHR-AGENT-006).
func RuntimeContext(ctx context.Context, runtime model.Ref) (model.RuntimeContext, error) {
	id := common.RequestIDFromContext(ctx)
	if !identifier.MatchString(id) {
		return model.RuntimeContext{}, ErrNoRequestID
	}
	return model.RuntimeContext{RuntimeInstanceID: runtime, ExecutionContextID: id}, nil
}
