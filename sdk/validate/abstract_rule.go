package validate

import (
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// AbstractRule carries identity and traceability and offers shared helpers; concrete rules embed it and implement Validate.
type AbstractRule struct {
	id   string
	reqs []string
}

func (r AbstractRule) ID() string             { return r.id }
func (r AbstractRule) Requirements() []string { return r.reqs }

func (r AbstractRule) finding(namespace, docID, msg string) Finding {
	return Finding{RuleID: r.id, Requirements: r.reqs, Class: ClassSemantic, Namespace: namespace, DocumentID: docID, Message: msg}
}

func (r AbstractRule) actions(c *corpus.Corpus) []model.ActionRecord {
	var out []model.ActionRecord
	for _, d := range c.OfKind(model.KindActionRecord) {
		if a, err := corpus.Decode[model.ActionRecord](d); err == nil {
			out = append(out, a)
		}
	}
	return out
}
