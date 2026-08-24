package validate

import "github.com/nauticana/charter/sdk/corpus"

// Validator is anything that produces findings over a corpus: the structural validator and every semantic rule.
type Validator interface {
	Validate(c *corpus.Corpus) []Finding
}

// Rule is one active semantic conformance rule with a stable id and requirement citations.
type Rule interface {
	Validator
	ID() string
	Requirements() []string
}
