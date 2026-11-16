package derivation

import (
	"fmt"

	"github.com/nauticana/charter/sdk/model"
)

// Root names a source document to derive and the version label to read; an empty label reads the current version.
type Root struct {
	ID      string
	Version string
}

// Options place the copies of one source namespace in a target namespace.
type Options struct {
	SourceNamespace string
	TargetNamespace string
	// Enterprise is the target enterprise every enterprise-scoped copy belongs to.
	Enterprise model.Ref
	// SourceVersion is recorded for a source whose kind declares no version, such as the source set's release.
	SourceVersion string
	// Upgrade replaces a copy derived from another version or content of its source; otherwise that copy is reused.
	Upgrade bool
}

// check refuses blank or equal namespaces, no roots, a blank root, and one root named at two versions.
func (o Options) check(roots []Root) error {
	switch {
	case o.SourceNamespace == "" || o.TargetNamespace == "":
		return fmt.Errorf("%w: source and target namespaces are required", ErrInvalidRequest)
	case o.SourceNamespace == o.TargetNamespace:
		return fmt.Errorf("%w: source and target namespace are both %s", ErrInvalidRequest, o.SourceNamespace)
	case len(roots) == 0:
		return fmt.Errorf("%w: no roots", ErrInvalidRequest)
	}
	versions := make(map[string]string, len(roots))
	for _, r := range roots {
		if r.ID == "" {
			return fmt.Errorf("%w: root without an identifier", ErrInvalidRequest)
		}
		if v, dup := versions[r.ID]; dup && v != r.Version {
			return fmt.Errorf("%w: root %s at versions %q and %q", ErrInvalidRequest, r.ID, v, r.Version)
		}
		versions[r.ID] = r.Version
	}
	return nil
}

// dropSourceEvidence removes the evidence a copy would cite in the source set; evidence elsewhere stays cited.
func (o Options) dropSourceEvidence(value map[string]any) {
	cited, _ := value["sourceEvidence"].([]any)
	var kept []any
	for _, item := range cited {
		if n, ok := item.(map[string]any); ok && n["namespace"] != nil && n["namespace"] != o.SourceNamespace {
			kept = append(kept, item)
		}
	}
	if len(kept) == 0 {
		delete(value, "sourceEvidence")
		return
	}
	value["sourceEvidence"] = kept
}
