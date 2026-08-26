package agent

import "context"

type Result string

const (
	Admitted Result = "admitted"
	Refused  Result = "refused"
	Error    Result = "error"
)

// Decision is the fail-closed outcome of an admission check and the requirement it enforces.
type Decision struct {
	Result      Result
	Reason      string
	Requirement string
}

// Admission decides whether a runtime may accept work in an execution context (CHR-AGENT-005).
type Admission interface {
	Admit(ctx context.Context, ec ExecutionContext) Decision
}
