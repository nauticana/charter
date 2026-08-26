package agent

import (
	"time"

	"github.com/nauticana/charter/sdk/model"
)

// ExecutionContext attributes work to a stable identity, the runtime instance and definition version performing it,
// and the single assignment in whose context authority is evaluated (CHR-AGENT-006, CHR-AGENT-007).
type ExecutionContext struct {
	Namespace          string
	Identity           model.Ref
	Runtime            model.Ref
	ExecutionContextID string
	Definition         model.Ref
	DefinitionVersion  string
	Assignment         model.Ref
	Participation      string
	Capability         *model.Ref
	At                 time.Time
}

func (c ExecutionContext) Actor() model.ObjectRef {
	return model.ObjectRef{Kind: model.KindAgentIdentity, ID: c.Identity.ID, Namespace: c.Identity.Namespace}
}

func (c ExecutionContext) RuntimeContext() model.RuntimeContext {
	return model.RuntimeContext{RuntimeInstanceID: c.Runtime, ExecutionContextID: c.ExecutionContextID}
}
