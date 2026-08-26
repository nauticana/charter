package evidence

import "regexp"

// Redactor removes secrets from evidence before it is appended or published (CHR-SEC-005, CHR-SEC-010).
type Redactor interface {
	Redact(raw []byte) ([]byte, error)
}

// BaseRedactor masks bearer tokens and key=value or key: value credentials inside JSON string values; it is a floor,
// not a substitute for keeping secrets out of reasons and payloads in the first place.
type BaseRedactor struct {
	Patterns []*regexp.Regexp
}

var _ Redactor = BaseRedactor{}

var defaultRedactions = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(bearer\s+)[^"\\\s]+`),
	regexp.MustCompile(`(?i)((?:password|passwd|secret|token|api[_-]?key|authorization|client[_-]?secret)\s*[=:]\s*)[^"\\\s,;]+`),
}

const redacted = "[redacted]"

func (r BaseRedactor) Redact(raw []byte) ([]byte, error) {
	patterns := r.Patterns
	if len(patterns) == 0 {
		patterns = defaultRedactions
	}
	out := raw
	for _, p := range patterns {
		out = p.ReplaceAll(out, []byte("${1}"+redacted))
	}
	return out, nil
}
