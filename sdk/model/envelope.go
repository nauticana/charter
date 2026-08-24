package model

import "encoding/json"

// Envelope carries the properties every Charter document has; typed documents embed it.
type Envelope struct {
	CharterSpecVersion string                     `json:"charterSpecVersion"`
	Namespace          string                     `json:"namespace"`
	ID                 string                     `json:"id"`
	Kind               Kind                       `json:"kind"`
	Name               string                     `json:"name,omitempty"`
	Description        string                     `json:"description,omitempty"`
	LifecycleState     string                     `json:"lifecycleState,omitempty"`
	Validity           *Validity                  `json:"validity,omitempty"`
	EnterpriseID       *Ref                       `json:"enterpriseId,omitempty"`
	Extensions         map[string]json.RawMessage `json:"extensions,omitempty"`
}
