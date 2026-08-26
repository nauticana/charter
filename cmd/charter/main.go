// Command charter validates Charter documents, runs the conformance manifest, and generates conformance claims.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sort"
	"time"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/validate"
)

const (
	exitOK      = 0
	exitFailed  = 1
	exitUsage   = 2
	defaultConf = "conformance"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return exitUsage
	}
	commands := map[string]func([]string, io.Writer, io.Writer) int{
		"validate": validateCmd, "conformance": conformanceCmd, "claim": claimCmd, "version": versionCmd,
	}
	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		usage(stdout)
		return exitOK
	}
	cmd, ok := commands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "charter: unknown command %q\n\n", args[0])
		usage(stderr)
		return exitUsage
	}
	return cmd(args[1:], stdout, stderr)
}

func usage(w io.Writer) {
	fmt.Fprint(w, `usage: charter <command> [flags]

  validate <dir>...   validate instance documents structurally and against the semantic rules
  conformance         run every fixture of the conformance manifest
  claim               run the manifest and print a ConformanceClaim document
  version             print the supported specification, schema catalog, and rule versions

Run "charter <command> -h" for the command's flags. Exit status: 0 conforming, 1 findings or failures, 2 usage or error.
`)
}

func validateCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "report format: text or json")
	conf := fs.String("conformance", "", "conformance directory; runs the active rules of -profile instead of every implemented rule")
	profile := fs.String("profile", "core-model", "profile whose active rules run when -conformance is set")
	specVersion := fs.String("spec-version", "", "specification version the documents must target (default: the embedded schemas' version)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "charter validate: at least one document directory is required")
		return exitUsage
	}
	meta, err := (validate.Catalog{}).Meta()
	if err != nil {
		return fail(stderr, err)
	}
	if *specVersion == "" {
		*specVersion = meta.SpecVersion
	}
	structural, err := validate.NewStructural()
	if err != nil {
		return fail(stderr, err)
	}
	rules, err := validate.NewRuleSet()
	if err != nil {
		return fail(stderr, err)
	}
	if *conf != "" {
		if rules, err = activeRules(rules, *conf, *profile); err != nil {
			return fail(stderr, err)
		}
	}
	c, err := loadDirs(fs.Args())
	if err != nil {
		return fail(stderr, err)
	}
	report := validationReport{SpecVersion: *specVersion, CatalogVersion: meta.CatalogVersion, Documents: c.Len(), Rules: ruleIDs(rules)}
	if *conf != "" {
		report.Profile = *profile
	}
	report.Findings = append(report.Findings, (validate.SpecVersion{Version: *specVersion}).Validate(c)...)
	report.Findings = append(report.Findings, structural.Validate(c)...)
	if len(report.Findings) > 0 {
		report.SemanticSkipped = true
	} else {
		for _, id := range report.Rules {
			report.Findings = append(report.Findings, rules[id].Validate(c)...)
		}
	}
	report.Result = model.ResultConforming
	if len(report.Findings) > 0 {
		report.Result = model.ResultNonconforming
	}
	if err := writeValidation(stdout, *format, report); err != nil {
		return fail(stderr, err)
	}
	if report.Result != model.ResultConforming {
		return exitFailed
	}
	return exitOK
}

func conformanceCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("conformance", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "report format: text or json")
	conf := fs.String("conformance", defaultConf, "conformance directory holding manifest.yaml")
	subject := fs.String("subject", "reference", "runtime and adapter under test for behavioral rules: reference (the SDK) or none")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	runner, err := newRunner(*conf, *subject)
	if err != nil {
		return fail(stderr, err)
	}
	manifest, err := runner.Manifest.Manifest()
	if err != nil {
		return fail(stderr, err)
	}
	results, err := runner.RunManifest()
	if err != nil {
		return fail(stderr, err)
	}
	report := conformanceReport{SpecVersion: manifest.Specification.Version, Fixtures: results, Result: model.ResultConforming}
	for _, r := range results {
		if !r.Passed {
			report.Result = model.ResultNonconforming
		}
	}
	if err := writeConformance(stdout, *format, report); err != nil {
		return fail(stderr, err)
	}
	if report.Result != model.ResultConforming {
		return exitFailed
	}
	return exitOK
}

func claimCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("claim", flag.ContinueOnError)
	fs.SetOutput(stderr)
	conf := fs.String("conformance", defaultConf, "conformance directory holding manifest.yaml")
	subject := fs.String("subject", "reference", "runtime and adapter under test for behavioral rules: reference (the SDK) or none")
	spec := validate.ClaimSpec{Implementation: model.ObjectRef{External: true}}
	fs.StringVar(&spec.Profile, "profile", "core-model", "conformance profile the claim targets")
	fs.StringVar(&spec.Namespace, "namespace", "", "namespace of the claim document (required)")
	fs.StringVar(&spec.ID, "id", "", "identifier of the claim document (required)")
	fs.StringVar(&spec.Name, "name", "", "display name of the claim document")
	fs.StringVar((*string)(&spec.Implementation.Kind), "implementation-kind", "GoModule", "kind of the external implementation reference")
	fs.StringVar(&spec.Implementation.ID, "implementation", "", "identifier of the implementation under test (required)")
	fs.StringVar(&spec.ImplementationVersion, "implementation-version", "", "version of the implementation under test (required)")
	date := fs.String("date", time.Now().UTC().Format("2006-01-02"), "result date, YYYY-MM-DD")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if spec.Namespace == "" || spec.ID == "" || spec.Implementation.ID == "" || spec.ImplementationVersion == "" {
		fmt.Fprintln(stderr, "charter claim: -namespace, -id, -implementation, and -implementation-version are required")
		return exitUsage
	}
	spec.ResultDate = model.Date(*date)
	runner, err := newRunner(*conf, *subject)
	if err != nil {
		return fail(stderr, err)
	}
	claim, err := runner.Claim(spec)
	if err != nil {
		return fail(stderr, err)
	}
	if err := writeJSON(stdout, claim); err != nil {
		return fail(stderr, err)
	}
	if claim.Result != model.ResultConforming {
		return exitFailed
	}
	return exitOK
}

func versionCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "report format: text or json")
	conf := fs.String("conformance", "", "conformance directory; reports its manifest version and active rules")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	meta, err := (validate.Catalog{}).Meta()
	if err != nil {
		return fail(stderr, err)
	}
	rules, err := validate.NewRuleSet()
	if err != nil {
		return fail(stderr, err)
	}
	v := versionReport{Charter: moduleVersion(), SpecVersion: meta.SpecVersion, CatalogVersion: meta.CatalogVersion, ImplementedRules: ruleIDs(rules)}
	if *conf != "" {
		manifest, err := (validate.ManifestReader{Dir: *conf}).Manifest()
		if err != nil {
			return fail(stderr, err)
		}
		v.ManifestSpecVersion = manifest.Specification.Version
		for _, p := range manifest.Profiles {
			if p.Status == "active" {
				v.ActiveProfiles = append(v.ActiveProfiles, p.ID)
			}
		}
		for _, r := range manifest.Rules {
			if r.Status == "active" {
				v.ActiveRules = append(v.ActiveRules, r.ID)
			}
		}
	}
	if err := writeVersion(stdout, *format, v); err != nil {
		return fail(stderr, err)
	}
	return exitOK
}

func newRunner(conf, subject string) (*validate.Runner, error) {
	runner, err := validate.NewRunner(conf)
	if err != nil {
		return nil, err
	}
	switch subject {
	case "reference":
	case "none":
		runner.Subject, runner.Adapter = nil, nil
	default:
		return nil, fmt.Errorf("unknown subject %q; use reference or none", subject)
	}
	return runner, nil
}

// activeRules keeps the implemented semantic rules an active profile lists; behavioral rules do not apply to documents,
// and a listed semantic rule without an implementation is an error.
func activeRules(all validate.RuleSet, conf, profile string) (validate.RuleSet, error) {
	reader := validate.ManifestReader{Dir: conf}
	manifest, err := reader.Manifest()
	if err != nil {
		return nil, err
	}
	paths := map[string]string{}
	for _, entry := range manifest.Rules {
		paths[entry.ID] = entry.Path
	}
	for _, p := range manifest.Profiles {
		if p.ID != profile {
			continue
		}
		if p.Status != "active" {
			return nil, fmt.Errorf("profile %s is %s", profile, p.Status)
		}
		out := validate.RuleSet{}
		for _, id := range p.Rules {
			def, err := reader.RuleDefinition(paths[id])
			if err != nil {
				return nil, err
			}
			if def.Verification != string(validate.ClassSemantic) {
				continue
			}
			rule, ok := all[id]
			if !ok {
				return nil, fmt.Errorf("profile %s lists rule %s, which this build does not implement", profile, id)
			}
			out[id] = rule
		}
		return out, nil
	}
	return nil, fmt.Errorf("unknown profile %s", profile)
}

func loadDirs(dirs []string) (*corpus.Corpus, error) {
	c := corpus.New()
	for _, dir := range dirs {
		part, err := corpus.NewDirLoader(dir).Load()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", dir, err)
		}
		for _, d := range part.Documents() {
			if err := c.Add(d); err != nil {
				return nil, fmt.Errorf("%s: %w", dir, err)
			}
		}
	}
	return c, nil
}

func ruleIDs(rules validate.RuleSet) []string {
	ids := make([]string, 0, len(rules))
	for id := range rules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func moduleVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(devel)"
}

func fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "charter: %v\n", err)
	return exitUsage
}
