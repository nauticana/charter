package authority

import (
	"sort"
	"strings"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// SubjectScope keys the business scope of an action by its subjects, so constrained actions on the same subjects conflict.
func SubjectScope(ownerNamespace string, subjects []model.ObjectRef) string {
	keys := make([]string, 0, len(subjects))
	for _, s := range subjects {
		keys = append(keys, string(s.Kind)+":"+corpus.ObjectKeyOf(ownerNamespace, s).String())
	}
	sort.Strings(keys)
	return strings.Join(keys, ";")
}

// Performed reports whether a recorded action was actually carried out: attempts whose authority or approval
// evaluation failed never happened and do not count for separation of duties.
func Performed(a model.ActionRecord) bool {
	for _, ev := range a.AuthorityEvaluations {
		if ev.Result != model.AuthorityAllowed {
			return false
		}
	}
	for _, ev := range a.ApprovalEvaluations {
		if ev.Result != model.ApprovalApproved && ev.Result != model.ApprovalNotRequired {
			return false
		}
	}
	return true
}
