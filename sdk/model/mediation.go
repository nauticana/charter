package model

import "time"

const (
	StageModelInvocation = "model-invocation"
	StageModelResult     = "model-result"
	StageToolInvocation  = "tool-invocation"
	StageToolResult      = "tool-result"
	StageRetrieval       = "retrieval"
	StageMemoryRead      = "memory-read"
	StageMemoryWrite     = "memory-write"
	StageSessionStart    = "session-start"
	StageSessionEnd      = "session-end"
	StageTurnStart       = "turn-start"
	StageTurnEnd         = "turn-end"
)

const (
	EnforcementEnforce       = "enforce"
	EnforcementShadow        = "shadow"
	ObligationTargetAct      = "act"
	ObligationTargetDecision = "decision"
)

const (
	OutcomeAllow      = "allow"
	OutcomeDeny       = "deny"
	OutcomeModify     = "modify"
	OutcomeAsk        = "ask"
	OutcomeDefer      = "defer"
	OutcomeNotDecided = "not-decided"
)

const (
	FailureFailClosed = "fail-closed"
	FailureFailOpen   = "fail-open"
	FailureObserve    = "observe"
)

const (
	MediatorAvailable       = "available"
	MediatorUnavailable     = "unavailable"
	MediatorBudgetExceeded  = "budget-exceeded"
	MediatorUndecidable     = "undecidable"
	ProvenanceAuthenticated = "authenticated"
	ProvenanceAsserted      = "asserted"
	EffectTightened         = "tightened"
	EffectRelaxed           = "relaxed"
	EffectNone              = "none"
	PermittedTightenOnly    = "tighten-only"
	PermittedRelaxOrTighten = "relax-or-tighten"
)

type Mediator struct {
	Party      string `json:"party"`
	MediatorID string `json:"mediatorId"`
	Name       string `json:"name,omitempty"`
}

type MediationScope struct {
	AgentIdentityIDs []Ref `json:"agentIdentityIds,omitempty"`
	CapabilityIDs    []Ref `json:"capabilityIds,omitempty"`
	InformationIDs   []Ref `json:"informationIds,omitempty"`
}

type DecisionBudget struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type MediationExclusion struct {
	Subject ObjectRef `json:"subject"`
	Reason  string    `json:"reason"`
}

type MediationCoverage struct {
	SubmittedShare float64              `json:"submittedShare"`
	Exclusions     []MediationExclusion `json:"exclusions,omitempty"`
}

type AttributeTrust struct {
	Attribute       string `json:"attribute"`
	Provenance      string `json:"provenance"`
	PermittedEffect string `json:"permittedEffect"`
}

type MediationEvidenceObligation struct {
	Retention      string `json:"retention,omitempty"`
	InformationIDs []Ref  `json:"informationIds,omitempty"`
}

type MediationProfile struct {
	Envelope
	Stage              string                      `json:"stage"`
	Mediator           Mediator                    `json:"mediator"`
	Scope              MediationScope              `json:"scope"`
	DeclaredOutcomes   []string                    `json:"declaredOutcomes"`
	EnforcementMode    string                      `json:"enforcementMode"`
	FailurePolicy      string                      `json:"failurePolicy"`
	DecisionBudget     DecisionBudget              `json:"decisionBudget"`
	Coverage           MediationCoverage           `json:"coverage"`
	AttributeTrust     []AttributeTrust            `json:"attributeTrust"`
	EvidenceObligation MediationEvidenceObligation `json:"evidenceObligation"`
	BoundaryControlIDs []Ref                       `json:"boundaryControlIds,omitempty"`
}

// Declares reports whether the profile may issue outcome (CHR-MED-003).
func (p MediationProfile) Declares(outcome string) bool {
	for _, o := range p.DeclaredOutcomes {
		if o == outcome {
			return true
		}
	}
	return false
}

type MediationRuntimeContext struct {
	RuntimeInstanceID  *Ref   `json:"runtimeInstanceId,omitempty"`
	ExecutionContextID string `json:"executionContextId"`
}

type MediationSubject struct {
	CapabilityIDs  []Ref `json:"capabilityIds"`
	InformationIDs []Ref `json:"informationIds"`
}

type AppliedObligation struct {
	Obligation        string `json:"obligation"`
	AppliedTo         string `json:"appliedTo"`
	TargetKind        string `json:"targetKind"`
	AuthorityExpanded bool   `json:"authorityExpanded"`
}

type MediationHandoff struct {
	Destination string `json:"destination"`
	Request     string `json:"request"`
}

type DecisionAttribute struct {
	Attribute  string `json:"attribute"`
	Provenance string `json:"provenance"`
	Effect     string `json:"effect"`
}

type MediationDecision struct {
	Envelope
	MediationProfileID   Ref                     `json:"mediationProfileId"`
	SubmissionID         string                  `json:"submissionId"`
	Actor                ObjectRef               `json:"actor"`
	Subject              MediationSubject        `json:"subject"`
	RuntimeContext       MediationRuntimeContext `json:"runtimeContext"`
	Stage                string                  `json:"stage"`
	DecidedAt            time.Time               `json:"decidedAt"`
	ActEffective         bool                    `json:"actEffective"`
	Enforced             bool                    `json:"enforced"`
	Outcome              string                  `json:"outcome"`
	MediatorAvailability string                  `json:"mediatorAvailability"`
	AppliedFailurePolicy string                  `json:"appliedFailurePolicy,omitempty"`
	NotDecidedReason     string                  `json:"notDecidedReason,omitempty"`
	SubjectReason        string                  `json:"subjectReason,omitempty"`
	AuditReference       string                  `json:"auditReference,omitempty"`
	ObligationsApplied   []AppliedObligation     `json:"obligationsApplied,omitempty"`
	ObligationsProposed  []AppliedObligation     `json:"obligationsProposed,omitempty"`
	Handoff              *MediationHandoff       `json:"handoff,omitempty"`
	DecisionAttributes   []DecisionAttribute     `json:"decisionAttributes,omitempty"`
	Supersedes           *Ref                    `json:"supersedes,omitempty"`
	ActionRecordID       *Ref                    `json:"actionRecordId,omitempty"`
	EvidenceRecordIDs    []Ref                   `json:"evidenceRecordIds,omitempty"`
}
