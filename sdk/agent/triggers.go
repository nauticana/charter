package agent

import "github.com/nauticana/charter/sdk/model"

// Triggered reports whether a definition declares the trigger an event binding maps to (CHR-AGENT-002).
func Triggered(def model.AgentDefinition, trigger string) bool {
	for _, t := range def.Triggers {
		if t == trigger {
			return true
		}
	}
	return false
}
