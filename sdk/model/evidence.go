package model

import "time"

type RuntimeContext struct {
	RuntimeInstanceID  Ref    `json:"runtimeInstanceId"`
	ExecutionContextID string `json:"executionContextId"`
}

const (
	AuthorityAllowed = "allowed"
	AuthorityDenied  = "denied"
	AuthorityMissing = "missing"
	AuthorityError   = "error"
)

type AuthorityEvaluation struct {
	AuthorityGrantID *Ref   `json:"authorityGrantId,omitempty"`
	Result           string `json:"result"`
	Reason           string `json:"reason"`
}

const (
	ApprovalApproved    = "approved"
	ApprovalRejected    = "rejected"
	ApprovalNotRequired = "not-required"
	ApprovalMissing     = "missing"
	ApprovalStale       = "stale"
)

const (
	DispositionExecuted      = "executed"
	DispositionBusinessError = "business-error"
	DispositionDenied        = "denied"
	DispositionFailed        = "failed"
	DispositionUnknown       = "unknown"
)

const (
	InformationAllowed = "allowed"
	InformationDenied  = "denied"
	InformationError   = "error"
)

type InformationEvaluation struct {
	InformationID Ref    `json:"informationId"`
	Purpose       string `json:"purpose"`
	Result        string `json:"result"`
	Reason        string `json:"reason"`
}

type ApprovalEvaluation struct {
	ApprovalID     *Ref   `json:"approvalId,omitempty"`
	ApprovedAction string `json:"approvedAction,omitempty"`
	Result         string `json:"result"`
	Reason         string `json:"reason"`
}

const (
	PostconditionSatisfied = "satisfied"
	PostconditionViolated  = "violated"
	PostconditionUnknown   = "unknown"
)

// PostconditionEvaluation is the observed state of one declared postcondition; EvidenceRecordIDs name the observed-fact
// records supporting it (CHR-EVID-011).
type PostconditionEvaluation struct {
	PostconditionID   string    `json:"postconditionId"`
	Result            string    `json:"result"`
	Reason            string    `json:"reason"`
	ObservedAt        time.Time `json:"observedAt"`
	EvidenceRecordIDs []Ref     `json:"evidenceRecordIds,omitempty"`
}

// PreconditionEvaluation is the state of one predicate precondition against the observation taken for the attempt (CHR-CAP-012).
type PreconditionEvaluation struct {
	PreconditionID    string    `json:"preconditionId"`
	Result            string    `json:"result"`
	Reason            string    `json:"reason"`
	ObservedAt        time.Time `json:"observedAt"`
	EvidenceRecordIDs []Ref     `json:"evidenceRecordIds,omitempty"`
}

// Effect classifies the required postconditions observed for an outcome (CHR-EVID-016).
const (
	EffectComplete   = "complete"
	EffectPartial    = "partial"
	EffectAbsent     = "absent"
	EffectUnobserved = "unobserved"
)

// Why an unobserved effect is unknown.
const (
	UnknownObservationFailed = "observation-failed"
	UnknownUnevidenced       = "unevidenced"
	UnknownContradictory     = "contradictory"
)

type ActionRecord struct {
	Envelope
	Actor                    ObjectRef                 `json:"actor"`
	RuntimeContext           RuntimeContext            `json:"runtimeContext"`
	AssignmentID             *Ref                      `json:"assignmentId,omitempty"`
	AssignmentBasis          string                    `json:"assignmentBasis,omitempty"`
	ResponsibilityID         *Ref                      `json:"responsibilityId,omitempty"`
	CapabilityID             Ref                       `json:"capabilityId"`
	MaterialInputsDigest     string                    `json:"materialInputsDigest,omitempty"`
	SubjectRefs              []ObjectRef               `json:"subjectRefs,omitempty"`
	OperationClass           string                    `json:"operationClass"`
	ActionTime               time.Time                 `json:"actionTime"`
	Outcome                  string                    `json:"outcome"`
	Disposition              string                    `json:"disposition"`
	AuthorityEvaluations     []AuthorityEvaluation     `json:"authorityEvaluations,omitempty"`
	ApprovalEvaluations      []ApprovalEvaluation      `json:"approvalEvaluations,omitempty"`
	InformationEvaluations   []InformationEvaluation   `json:"informationEvaluations,omitempty"`
	ApprovalIDs              []Ref                     `json:"approvalIds,omitempty"`
	EvidenceRecordIDs        []Ref                     `json:"evidenceRecordIds,omitempty"`
	PreconditionEvaluations  []PreconditionEvaluation  `json:"preconditionEvaluations,omitempty"`
	PostconditionEvaluations []PostconditionEvaluation `json:"postconditionEvaluations,omitempty"`
	Effect                   string                    `json:"effect,omitempty"`
	UnknownCause             string                    `json:"unknownCause,omitempty"`
	// ReconcilesActionID names the earlier attempt whose unknown outcome this record resolves (CHR-EVID-012).
	ReconcilesActionID        *Ref                       `json:"reconcilesActionId,omitempty"`
	SodExceptionID            *Ref                       `json:"sodExceptionId,omitempty"`
	OperatingLimitEvaluations []OperatingLimitEvaluation `json:"operatingLimitEvaluations,omitempty"`
}

const (
	LimitWithin   = "within"
	LimitExceeded = "exceeded"
)

type OperatingLimitEvaluation struct {
	LimitKind string `json:"limitKind"`
	Result    string `json:"result"`
	Reason    string `json:"reason"`
}

const (
	CategoryObservedFact      = "observed-fact"
	CategoryAgentAssertion    = "agent-assertion"
	CategoryHumanDecision     = "human-decision"
	CategoryExternalResponse  = "external-response"
	CategoryDerivedConclusion = "derived-conclusion"
)

type EvidenceRecord struct {
	Envelope
	Category        string    `json:"category"`
	Content         string    `json:"content"`
	RecordedAt      time.Time `json:"recordedAt"`
	Sources         []string  `json:"sources,omitempty"`
	Transformations []string  `json:"transformations,omitempty"`
	Supersedes      *Ref      `json:"supersedes,omitempty"`
	// Integrity digests Content, so one immutable record is verifiable outside a bundle (CHR-EVID-005).
	Integrity  *Integrity  `json:"integrity,omitempty"`
	Assessment *Assessment `json:"assessment,omitempty"`
	Fact       *Fact       `json:"fact,omitempty"`
}

// Fact is the value of one attribute of a subject at a time, as its source stated it (CHR-EVID-014).
type Fact struct {
	Subject         ObjectRef `json:"subject"`
	InformationID   Ref       `json:"informationId"`
	Attribute       string    `json:"attribute"`
	Value           any       `json:"value"`
	ValidAt         time.Time `json:"validAt"`
	SystemProfileID *Ref      `json:"systemProfileId,omitempty"`
}

const (
	AssessmentEvaluation = "evaluation"
	AssessmentFeedback   = "feedback"
)

// Assessment is an evaluation that justified releasing a definition or feedback on an outcome (CHR-EVID-013).
type Assessment struct {
	AssessmentKind string    `json:"assessmentKind"`
	Subject        ObjectRef `json:"subject"`
	SubjectDigest  string    `json:"subjectDigest,omitempty"`
	Judge          ObjectRef `json:"judge"`
	JudgeVersion   string    `json:"judgeVersion,omitempty"`
	CaseSet        *CaseSet  `json:"caseSet,omitempty"`
	Scores         []Score   `json:"scores,omitempty"`
	Rating         string    `json:"rating,omitempty"`
	Reason         string    `json:"reason,omitempty"`
}

type CaseSet struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type Score struct {
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
}

type Integrity struct {
	Method     string     `json:"method"`
	Value      string     `json:"value,omitempty"`
	VerifiedAt *time.Time `json:"verifiedAt,omitempty"`
}

type EvidenceBundle struct {
	Envelope
	Subject          ObjectRef `json:"subject"`
	RecordIDs        []Ref     `json:"recordIds"`
	AssuranceProfile string    `json:"assuranceProfile"`
	Integrity        Integrity `json:"integrity"`
}

type ExceptionRecord struct {
	Envelope
	ViolatedExpectation string    `json:"violatedExpectation"`
	Affected            ObjectRef `json:"affected"`
	Disposition         string    `json:"disposition"`
	EscalationTarget    ObjectRef `json:"escalationTarget"`
}

type Escalation struct {
	Envelope
	Reason               string    `json:"reason"`
	Recipient            ObjectRef `json:"recipient"`
	RequestedDecision    string    `json:"requestedDecision"`
	Urgency              string    `json:"urgency"`
	ResultingDisposition string    `json:"resultingDisposition,omitempty"`
}
