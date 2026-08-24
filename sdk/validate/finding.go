// Package validate provides structural validation, semantic conformance rules, and fixture execution.
package validate

type Class string

const (
	ClassStructural Class = "structural"
	ClassSemantic   Class = "semantic"
)

// Finding is one requirement-linked validation failure.
type Finding struct {
	RuleID       string
	Requirements []string
	Class        Class
	Namespace    string
	DocumentID   string
	Path         string
	Message      string
}
