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

type ActionRecord struct {
	Envelope
	Actor                  ObjectRef               `json:"actor"`
	RuntimeContext         RuntimeContext          `json:"runtimeContext"`
	AssignmentID           *Ref                    `json:"assignmentId,omitempty"`
	ResponsibilityID       *Ref                    `json:"responsibilityId,omitempty"`
	CapabilityID           Ref                     `json:"capabilityId"`
	MaterialInputsDigest   string                  `json:"materialInputsDigest,omitempty"`
	SubjectRefs            []ObjectRef             `json:"subjectRefs,omitempty"`
	OperationClass         string                  `json:"operationClass"`
	ActionTime             time.Time               `json:"actionTime"`
	Outcome                string                  `json:"outcome"`
	Disposition            string                  `json:"disposition"`
	AuthorityEvaluations   []AuthorityEvaluation   `json:"authorityEvaluations,omitempty"`
	ApprovalEvaluations    []ApprovalEvaluation    `json:"approvalEvaluations,omitempty"`
	InformationEvaluations []InformationEvaluation `json:"informationEvaluations,omitempty"`
	ApprovalIDs            []Ref                   `json:"approvalIds,omitempty"`
	EvidenceRecordIDs      []Ref                   `json:"evidenceRecordIds,omitempty"`
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
