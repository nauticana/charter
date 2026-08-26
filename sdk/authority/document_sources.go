package authority

import (
	"context"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// DocumentGrantSource serves grants from any corpus.Source.
type DocumentGrantSource struct {
	Documents corpus.AbstractDocumentProvider
}

var _ GrantSource = (*DocumentGrantSource)(nil)

func NewDocumentGrantSource(s corpus.Source) *DocumentGrantSource {
	return &DocumentGrantSource{corpus.AbstractDocumentProvider{Source: s}}
}

func (s *DocumentGrantSource) Grants(ctx context.Context, req Request) ([]model.AuthorityGrant, error) {
	all, err := corpus.ListAs[model.AuthorityGrant](ctx, &s.Documents, model.KindAuthorityGrant)
	if err != nil {
		return nil, err
	}
	return (&BaseGrantSource{Items: all}).Grants(ctx, req)
}

// DocumentApprovalSource serves approvals from any corpus.Source.
type DocumentApprovalSource struct {
	Documents corpus.AbstractDocumentProvider
}

var _ ApprovalSource = (*DocumentApprovalSource)(nil)

func NewDocumentApprovalSource(s corpus.Source) *DocumentApprovalSource {
	return &DocumentApprovalSource{corpus.AbstractDocumentProvider{Source: s}}
}

func (s *DocumentApprovalSource) Approvals(ctx context.Context, req ApprovalRequest) ([]model.Approval, error) {
	all, err := corpus.ListAs[model.Approval](ctx, &s.Documents, model.KindApproval)
	if err != nil {
		return nil, err
	}
	return (&BaseApprovalSource{Items: all}).Approvals(ctx, req)
}
