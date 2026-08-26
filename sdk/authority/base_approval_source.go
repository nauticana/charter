package authority

import (
	"context"

	"github.com/nauticana/charter/sdk/model"
)

// BaseApprovalSource serves approvals for the requested action from memory.
type BaseApprovalSource struct {
	Items []model.Approval
}

var _ ApprovalSource = (*BaseApprovalSource)(nil)

func (s *BaseApprovalSource) Approvals(_ context.Context, req ApprovalRequest) ([]model.Approval, error) {
	var out []model.Approval
	for _, p := range s.Items {
		if p.ApprovedAction == req.ApprovedAction {
			out = append(out, p)
		}
	}
	return out, nil
}
