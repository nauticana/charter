package model

type InformationDefinition struct {
	Envelope
	BusinessMeaning     string    `json:"businessMeaning"`
	OwnerOrSteward      ObjectRef `json:"ownerOrSteward"`
	Classification      string    `json:"classification"`
	GovernancePolicyIDs []Ref     `json:"governancePolicyIds"`
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
