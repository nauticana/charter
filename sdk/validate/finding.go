// Package validate provides structural validation, semantic conformance rules, and fixture execution.
package validate

type Class string

const (
	ClassStructural Class = "structural"
	ClassSemantic   Class = "semantic"
	ClassBehavioral Class = "runtime-behavioral"
)

// Finding is one requirement-linked validation failure.
type Finding struct {
	RuleID       string   `json:"ruleId"`
	Requirements []string `json:"requirements,omitempty"`
	Class        Class    `json:"class"`
	Namespace    string   `json:"namespace,omitempty"`
	DocumentID   string   `json:"documentId,omitempty"`
	Path         string   `json:"path,omitempty"`
	Message      string   `json:"message"`
}
