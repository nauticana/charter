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
	// SourceEvidence is admitted by enterprise and process definition schemas only (CHR-INFO-009).
	SourceEvidence []Ref       `json:"sourceEvidence,omitempty"`
	DerivedFrom    *Provenance `json:"derivedFrom,omitempty"`
}

const (
	AdoptionAdopted  = "adopted"
	AdoptionTailored = "tailored"
)

// Provenance names the document in another document set a document was derived from (CHR-CONF-015, CHR-AGENT-010).
type Provenance struct {
	Source   ObjectRef `json:"source"`
	Version  string    `json:"version"`
	Digest   string    `json:"digest"`
	Adoption string    `json:"adoption,omitempty"`
}
