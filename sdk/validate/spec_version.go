package validate

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nauticana/charter/sdk/corpus"
)

// SpecVersion reports documents the release it validates against cannot read: another major version or a later release.
// A document keeps the version it was authored against; every later minor release of its major accepts it (CHR-CONF-001,
// CHR-CONF-010).
type SpecVersion struct {
	Version string
}

var _ Validator = SpecVersion{}

func (v SpecVersion) Validate(c *corpus.Corpus) []Finding {
	release, releaseOK := semver(v.Version)
	var out []Finding
	for _, d := range c.Documents() {
		doc, ok := semver(d.CharterSpecVersion)
		if ok && releaseOK && doc[0] == release[0] && !(doc[1] > release[1] || doc[1] == release[1] && doc[2] > release[2]) {
			continue
		}
		out = append(out, Finding{RuleID: "spec-version", Requirements: []string{"CHR-CONF-001"}, Class: ClassStructural, Namespace: d.Namespace, DocumentID: d.ID,
			Path: "/charterSpecVersion", Message: fmt.Sprintf("targets specification %q, which %s cannot read", d.CharterSpecVersion, v.Version)})
	}
	return out
}

func semver(s string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
