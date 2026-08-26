package agent

import (
	"context"
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/identity"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/organization"
)

// BaseAdmission admits work only when identity, runtime, and definition are mutually consistent and active,
// the single assignment covers the identity at the context time, and any capability is declared by the definition.
type BaseAdmission struct {
	Agents      Provider
	Assignments AssignmentSource
}

var _ Admission = (*BaseAdmission)(nil)

func (a *BaseAdmission) Admit(ctx context.Context, ec ExecutionContext) Decision {
	if a.Agents == nil || a.Assignments == nil {
		return Decision{Result: Error, Reason: "admission requires agent and assignment providers", Requirement: "CHR-AUTH-010"}
	}
	identityKey := corpus.KeyOf(ec.Namespace, ec.Identity)
	rt, err := a.Agents.Runtime(ctx, ec.Namespace, ec.Runtime)
	if err != nil {
		return failure(err, "CHR-AGENT-006")
	}
	if corpus.KeyOf(rt.Namespace, rt.AgentIdentityID) != identityKey {
		return refuse(fmt.Sprintf("runtime %s does not act as %s", rt.ID, ec.Identity.ID), "CHR-AGENT-006")
	}
	if corpus.KeyOf(rt.Namespace, rt.AgentDefinitionID) != corpus.KeyOf(ec.Namespace, ec.Definition) || rt.DefinitionVersion != ec.DefinitionVersion {
		return refuse(fmt.Sprintf("runtime %s operates %s version %s", rt.ID, rt.AgentDefinitionID.ID, rt.DefinitionVersion), "CHR-AGENT-006")
	}
	if rt.LifecycleState != model.LifecycleActive {
		return refuse(fmt.Sprintf("runtime %s is %s", rt.ID, rt.LifecycleState), "CHR-AGENT-005")
	}
	id, err := a.Agents.Identity(ctx, ec.Namespace, ec.Identity)
	if err != nil {
		return failure(err, "CHR-ID-001")
	}
	if err := (identity.Lifecycle{Current: id.LifecycleState, History: id.LifecycleHistory}).ActiveAt(ec.At); err != nil {
		return refuse(id.ID+": "+err.Error(), "CHR-ID-005")
	}
	def, err := a.Agents.Definition(ctx, ec.Namespace, ec.Definition)
	if err != nil {
		return failure(err, "CHR-AGENT-001")
	}
	if corpus.KeyOf(def.Namespace, def.AgentIdentityID) != identityKey {
		return refuse(fmt.Sprintf("definition %s belongs to %s", def.ID, def.AgentIdentityID.ID), "CHR-AGENT-006")
	}
	if def.LifecycleState != "" && def.LifecycleState != model.LifecycleActive {
		return refuse(fmt.Sprintf("definition %s is %s", def.ID, def.LifecycleState), "CHR-AGENT-005")
	}
	asg, err := a.Assignments.Assignment(ctx, ec.Namespace, ec.Assignment)
	if err != nil {
		return failure(err, "CHR-AGENT-007")
	}
	if asg.Subject.Kind != model.KindAgentIdentity || corpus.ObjectKeyOf(asg.Namespace, asg.Subject) != identityKey {
		return refuse(fmt.Sprintf("assignment %s belongs to %s %s", asg.ID, asg.Subject.Kind, asg.Subject.ID), "CHR-AGENT-007")
	}
	if asg.Participation == model.ParticipationOccupies {
		return refuse(fmt.Sprintf("assignment %s places an agent in a position", asg.ID), "CHR-ENT-011")
	}
	if !organization.Current(asg, ec.At) {
		return refuse(fmt.Sprintf("assignment %s is not effective at %s", asg.ID, ec.At.Format("2006-01-02")), "CHR-AGENT-007")
	}
	if ec.Participation != "" && ec.Participation != asg.Participation {
		return refuse(fmt.Sprintf("assignment %s permits %s, not %s", asg.ID, asg.Participation, ec.Participation), "CHR-PROC-006")
	}
	if ec.Capability != nil && !Declares(def, ec.Namespace, *ec.Capability) {
		return refuse(fmt.Sprintf("definition %s does not declare capability %s", def.ID, ec.Capability.ID), "CHR-AGENT-003")
	}
	return Decision{Result: Admitted, Reason: "identity, runtime, definition, and assignment are consistent and effective"}
}

// Declares reports whether a definition lists the capability (CHR-AGENT-003).
func Declares(def model.AgentDefinition, owner string, capability model.Ref) bool {
	want := corpus.KeyOf(owner, capability)
	for _, c := range def.CapabilityIDs {
		if corpus.KeyOf(def.Namespace, c) == want {
			return true
		}
	}
	return false
}

func refuse(reason, requirement string) Decision {
	return Decision{Result: Refused, Reason: reason, Requirement: requirement}
}

func failure(err error, requirement string) Decision {
	return Decision{Result: Error, Reason: err.Error(), Requirement: requirement}
}
