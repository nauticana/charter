package identity

import "github.com/nauticana/charter/sdk/model"

// Actor is a resolved identity, independent of its credentials, sessions, and runtime instances (CHR-ID-003).
type Actor struct {
	model.Envelope
	Lifecycle Lifecycle
}

// Ref returns the explicitly namespaced reference evidence should retain for the actor (CHR-ID-006).
func (a Actor) Ref() model.ObjectRef {
	return model.ObjectRef{Kind: a.Kind, ID: a.ID, Namespace: a.Namespace}
}
