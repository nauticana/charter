package validate

import (
	"fmt"

	"github.com/nauticana/charter/sdk/agent"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

type RuntimeActorRule struct{ AbstractRule }
type DeclaredCapabilityRule struct{ AbstractRule }

var (
	_ Rule = RuntimeActorRule{}
	_ Rule = DeclaredCapabilityRule{}
)

func (r RuntimeActorRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, a := range r.actions(c) {
		rt, ok := resolveAs[model.AgentRuntime](c, a.Namespace, a.RuntimeContext.RuntimeInstanceID, model.KindAgentRuntime)
		if a.Actor.Kind != model.KindAgentIdentity || !ok {
			continue
		}
		actor := corpus.ObjectKeyOf(a.Namespace, a.Actor)
		if corpus.KeyOf(rt.Namespace, rt.AgentIdentityID) != actor {
			out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("runtime %s acts as %s, not actor %s", rt.ID, rt.AgentIdentityID.ID, a.Actor.ID)))
		}
		if def, ok := resolveAs[model.AgentDefinition](c, rt.Namespace, rt.AgentDefinitionID, model.KindAgentDefinition); ok && corpus.KeyOf(def.Namespace, def.AgentIdentityID) != actor {
			out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("runtime %s operates definition %s of %s, not actor %s", rt.ID, def.ID, def.AgentIdentityID.ID, a.Actor.ID)))
		}
	}
	return out
}

func (r DeclaredCapabilityRule) Validate(c *corpus.Corpus) []Finding {
	var out []Finding
	for _, a := range r.actions(c) {
		rt, ok := resolveAs[model.AgentRuntime](c, a.Namespace, a.RuntimeContext.RuntimeInstanceID, model.KindAgentRuntime)
		if a.Actor.Kind != model.KindAgentIdentity || !ok {
			continue
		}
		def, ok := resolveAs[model.AgentDefinition](c, rt.Namespace, rt.AgentDefinitionID, model.KindAgentDefinition)
		if ok && !agent.Declares(def, a.Namespace, a.CapabilityID) {
			out = append(out, r.finding(a.Namespace, a.ID, fmt.Sprintf("definition %s does not declare capability %s", def.ID, a.CapabilityID.ID)))
		}
	}
	return out
}
