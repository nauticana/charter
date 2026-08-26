package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/nauticana/charter/sdk/validate"
)

type validationReport struct {
	SpecVersion     string             `json:"specVersion"`
	CatalogVersion  string             `json:"catalogVersion"`
	Profile         string             `json:"profile,omitempty"`
	Documents       int                `json:"documents"`
	Rules           []string           `json:"rules"`
	SemanticSkipped bool               `json:"semanticSkipped,omitempty"`
	Findings        []validate.Finding `json:"findings"`
	Result          string             `json:"result"`
}

type conformanceReport struct {
	SpecVersion string                   `json:"specVersion"`
	Fixtures    []validate.FixtureResult `json:"fixtures"`
	Result      string                   `json:"result"`
}

type versionReport struct {
	Charter             string   `json:"charter"`
	SpecVersion         string   `json:"specVersion"`
	CatalogVersion      string   `json:"catalogVersion"`
	ImplementedRules    []string `json:"implementedRules"`
	ManifestSpecVersion string   `json:"manifestSpecVersion,omitempty"`
	ActiveProfiles      []string `json:"activeProfiles,omitempty"`
	ActiveRules         []string `json:"activeRules,omitempty"`
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func checkFormat(format string) error {
	if format != "text" && format != "json" {
		return fmt.Errorf("unknown format %q; use text or json", format)
	}
	return nil
}

func writeValidation(w io.Writer, format string, r validationReport) error {
	if err := checkFormat(format); err != nil {
		return err
	}
	if format == "json" {
		return writeJSON(w, r)
	}
	fmt.Fprintf(w, "charter validate: %d documents against specification %s (schema catalog %s), %d rules\n", r.Documents, r.SpecVersion, r.CatalogVersion, len(r.Rules))
	for _, f := range r.Findings {
		fmt.Fprintln(w, findingLine(f))
	}
	if r.SemanticSkipped {
		fmt.Fprintln(w, "semantic rules skipped: documents are not schema-valid")
	}
	fmt.Fprintf(w, "result: %s (%d findings)\n", r.Result, len(r.Findings))
	return nil
}

func findingLine(f validate.Finding) string {
	where := f.Namespace + ":" + f.DocumentID + f.Path
	requirements := ""
	if len(f.Requirements) > 0 {
		requirements = " [" + strings.Join(f.Requirements, ", ") + "]"
	}
	return fmt.Sprintf("%-10s %-22s %s: %s%s", f.Class, f.RuleID, where, f.Message, requirements)
}

func writeConformance(w io.Writer, format string, r conformanceReport) error {
	if err := checkFormat(format); err != nil {
		return err
	}
	if format == "json" {
		return writeJSON(w, r)
	}
	fmt.Fprintf(w, "charter conformance: specification %s, %d fixtures\n", r.SpecVersion, len(r.Fixtures))
	for _, fx := range r.Fixtures {
		status := "PASS"
		if !fx.Passed {
			status = "FAIL"
		}
		fmt.Fprintf(w, "%s  %-8s %s", status, fx.Expected, fx.Dir)
		if fx.Detail != "" {
			fmt.Fprintf(w, ": %s", fx.Detail)
		}
		fmt.Fprintln(w)
		if !fx.Passed {
			for _, f := range fx.Findings {
				fmt.Fprintln(w, "      "+findingLine(f))
			}
		}
	}
	fmt.Fprintf(w, "result: %s\n", r.Result)
	return nil
}

func writeVersion(w io.Writer, format string, v versionReport) error {
	if err := checkFormat(format); err != nil {
		return err
	}
	if format == "json" {
		return writeJSON(w, v)
	}
	fmt.Fprintf(w, "charter %s\nspecification %s\nschema catalog %s\nimplemented rules %d\n", v.Charter, v.SpecVersion, v.CatalogVersion, len(v.ImplementedRules))
	if v.ManifestSpecVersion != "" {
		fmt.Fprintf(w, "manifest specification %s\nactive profiles %s\nactive rules %d\n", v.ManifestSpecVersion, strings.Join(v.ActiveProfiles, ", "), len(v.ActiveRules))
	}
	return nil
}
