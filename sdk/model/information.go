package model

type InformationDefinition struct {
	Envelope
	BusinessMeaning     string    `json:"businessMeaning"`
	OwnerOrSteward      ObjectRef `json:"ownerOrSteward"`
	Classification      string    `json:"classification"`
	GovernancePolicyIDs []Ref     `json:"governancePolicyIds"`
	// Shape types the information without making Charter a universal data model (CHR-INFO-010).
	Shape         *InformationShape `json:"shape,omitempty"`
	FactAuthority *FactAuthority    `json:"factAuthority,omitempty"`
}

const (
	CardinalityOne  = "one"
	CardinalityMany = "many"
)

type InformationShape struct {
	Properties    []ShapeProperty     `json:"properties"`
	Identity      []string            `json:"identity"`
	Relationships []ShapeRelationship `json:"relationships,omitempty"`
}

// ShapeProperty is one named property; Type is a JSON Schema type name.
type ShapeProperty struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	Required       bool   `json:"required,omitempty"`
	Classification string `json:"classification,omitempty"`
}

type ShapeRelationship struct {
	Name                string `json:"name"`
	TargetInformationID Ref    `json:"targetInformationId"`
	Cardinality         string `json:"cardinality"`
}

// FactAuthority ranks the sources of facts about the information, most authoritative first, and bounds their age
// (CHR-EVID-014, CHR-EVID-015).
type FactAuthority struct {
	Ranks         []FactSource `json:"ranks"`
	MaxAgeSeconds int64        `json:"maxAgeSeconds,omitempty"`
}

// FactSource names a system profile or an evidence category; exactly one is set.
type FactSource struct {
	SystemProfileID *Ref   `json:"systemProfileId,omitempty"`
	Category        string `json:"category,omitempty"`
}

const (
	ArtifactSourceData           = "source-data"
	ArtifactActionEvidence       = "action-evidence"
	ArtifactPrompt               = "prompt"
	ArtifactIntermediateArtifact = "intermediate-artifact"
	ArtifactCache                = "cache"
	ArtifactExternalReference    = "external-reference"
)

type RetentionRule struct {
	ArtifactKind  string `json:"artifactKind"`
	RetentionRule string `json:"retentionRule"`
	DeletionRule  string `json:"deletionRule"`
}

const (
	ConflictFailClosed = "fail-closed"
	ConflictPrecedence = "apply-declared-precedence"
)

type InformationGovernancePolicy struct {
	Envelope
	Scope                []ObjectRef     `json:"scope"`
	PermittedUses        []string        `json:"permittedUses"`
	AccessConstraints    []string        `json:"accessConstraints"`
	RetentionAndDeletion []RetentionRule `json:"retentionAndDeletion"`
	Precedence           int             `json:"precedence"`
	ConflictBehavior     string          `json:"conflictBehavior"`
}
