package capability

import (
	"context"
	"errors"
	"fmt"

	"github.com/nauticana/charter/sdk/agent"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/organization"
)

var ErrNoRecipient = errors.New("no accountable authority to escalate to")

// Escalator turns a stopped invocation into an exception and an escalation to the accountable authority
// (CHR-AGENT-004, CHR-EVID-006, CHR-EVID-007).
type Escalator interface {
	Applies(status Status) bool
	Escalate(ctx context.Context, inv Invocation, res Result) (model.ExceptionRecord, model.Escalation, error)
}

// RecipientResolver names who receives an escalation for an invocation.
type RecipientResolver interface {
	Recipient(ctx context.Context, inv Invocation) (model.ObjectRef, error)
}

// BaseEscalator escalates unknown and failed outcomes by default; denials are left to the runtime, which may request
// approval instead.
type BaseEscalator struct {
	Recipients RecipientResolver
	IDs        IDGenerator
	Statuses   []Status
}

var _ Escalator = (*BaseEscalator)(nil)

func (e *BaseEscalator) Applies(status Status) bool {
	statuses := e.Statuses
	if len(statuses) == 0 {
		statuses = []Status{StatusUnknown, StatusFailed}
	}
	for _, s := range statuses {
		if s == status {
			return true
		}
	}
	return false
}

func (e *BaseEscalator) Escalate(ctx context.Context, inv Invocation, res Result) (model.ExceptionRecord, model.Escalation, error) {
	if e.Recipients == nil || e.IDs == nil {
		return model.ExceptionRecord{}, model.Escalation{}, errors.New("escalator requires recipients and ids")
	}
	recipient, err := e.Recipients.Recipient(ctx, inv)
	if err != nil {
		return model.ExceptionRecord{}, model.Escalation{}, err
	}
	if res.Action == nil {
		return model.ExceptionRecord{}, model.Escalation{}, errors.New("escalation needs the recorded action")
	}
	specVersion := res.Action.CharterSpecVersion
	exception := model.ExceptionRecord{ViolatedExpectation: clip(fmt.Sprintf("declared outcome for %s: %s", inv.CapabilityID.ID, res.Reason)),
		Affected: affected(inv, res), Disposition: "open", EscalationTarget: recipient}
	exception.CharterSpecVersion, exception.Namespace, exception.ID, exception.Kind = specVersion, inv.Namespace, e.IDs.NewID(model.KindExceptionRecord), model.KindExceptionRecord
	escalation := model.Escalation{Reason: clip(res.Reason), Recipient: recipient, RequestedDecision: clip(requestedDecision(inv, res)), Urgency: urgency(res.Status)}
	escalation.CharterSpecVersion, escalation.Namespace, escalation.ID, escalation.Kind = specVersion, inv.Namespace, e.IDs.NewID(model.KindEscalation), model.KindEscalation
	if inv.EnterpriseID.ID != "" {
		enterprise := inv.EnterpriseID
		exception.EnterpriseID, escalation.EnterpriseID = &enterprise, &enterprise
	}
	return exception, escalation, nil
}

// affected names the process or task instance the action concerned, or else the action record itself.
func affected(inv Invocation, res Result) model.ObjectRef {
	for _, s := range inv.SubjectRefs {
		if s.Kind == model.KindProcessInstance || s.Kind == model.KindTaskInstance {
			return s
		}
	}
	if res.Action != nil {
		return model.ObjectRef{Kind: model.KindActionRecord, ID: res.Action.ID}
	}
	return model.ObjectRef{Kind: model.KindCapabilityContract, ID: inv.CapabilityID.ID, Namespace: inv.CapabilityID.Namespace}
}

func requestedDecision(inv Invocation, res Result) string {
	switch res.Status {
	case StatusUnknown:
		return fmt.Sprintf("confirm the external outcome of %s and reconcile idempotency key %s before any retry", inv.CapabilityID.ID, inv.IdempotencyKey)
	case StatusFailed:
		return fmt.Sprintf("decide whether %s is retried or abandoned", inv.CapabilityID.ID)
	}
	return fmt.Sprintf("review the %s outcome of %s (%s)", res.Status, inv.CapabilityID.ID, res.Requirement)
}

func urgency(status Status) string {
	if status == StatusUnknown {
		return "high"
	}
	return "normal"
}

// BaseAccountableRecipient escalates to the accountable authority of the acting agent's definition, or, for a human
// actor, of the responsibility the invocation names.
type BaseAccountableRecipient struct {
	Agents       agent.Provider
	Organization organization.Provider
}

var _ RecipientResolver = (*BaseAccountableRecipient)(nil)

func (r *BaseAccountableRecipient) Recipient(ctx context.Context, inv Invocation) (model.ObjectRef, error) {
	switch inv.Actor.Kind {
	case model.KindAgentIdentity:
		if r.Agents == nil {
			return model.ObjectRef{}, errors.New("agent escalation requires an agent provider")
		}
		rt, err := r.Agents.Runtime(ctx, inv.Namespace, inv.Runtime.RuntimeInstanceID)
		if err != nil {
			return model.ObjectRef{}, fmt.Errorf("agent runtime: %w", err)
		}
		def, err := r.Agents.Definition(ctx, rt.Namespace, rt.AgentDefinitionID)
		if err != nil {
			return model.ObjectRef{}, fmt.Errorf("agent definition: %w", err)
		}
		return namespaced(def.Namespace, def.Accountable), nil
	case model.KindHumanIdentity:
		if inv.ResponsibilityID == nil {
			return model.ObjectRef{}, fmt.Errorf("%w: human invocation names no responsibility", ErrNoRecipient)
		}
		if r.Organization == nil {
			return model.ObjectRef{}, errors.New("human escalation requires an organization provider")
		}
		resp, err := r.Organization.Responsibility(ctx, inv.Namespace, *inv.ResponsibilityID)
		if err != nil {
			return model.ObjectRef{}, fmt.Errorf("responsibility: %w", err)
		}
		return namespaced(resp.Namespace, resp.Accountable), nil
	}
	return model.ObjectRef{}, fmt.Errorf("%w: %s %s", ErrNoRecipient, inv.Actor.Kind, inv.Actor.ID)
}

func namespaced(owner string, ref model.ObjectRef) model.ObjectRef {
	if ref.Namespace == "" {
		ref.Namespace = owner
	}
	return ref
}
