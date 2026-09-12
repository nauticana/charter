// Package authority evaluates grants, limits, delegation depth, and separation of duties with fail-closed results.
package authority

import (
	"time"

	"github.com/nauticana/charter/sdk/model"
)

type Result string

const (
	Allowed Result = "allowed"
	Denied  Result = "denied"
	Missing Result = "missing"
	Error   Result = "error"
)

// Decision is the structured, evidence-ready outcome of an authority evaluation.
type Decision struct {
	GrantRef *model.Ref
	Result   Result
	Reason   string
}

// Measure is the observed value for one limitKind of the requested action.
type Measure struct {
	Value            any
	Unit             string
	Currency         string
	CurrencyExponent int
}

type Request struct {
	Namespace             string
	EnterpriseID          model.Ref
	Actor                 model.ObjectRef
	CapabilityID          model.Ref
	ResourceScope         string
	At                    time.Time
	OrganizationalContext string
	Measures              map[string]Measure
	AuthorityChain        model.AuthorityChain
}
