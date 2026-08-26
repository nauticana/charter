# Charter CLI

`charter` is the reference validation tool over the Go SDK: it validates instance documents, runs the conformance manifest, and generates conformance claims. It is a reference tool, not the exclusive means of establishing conformance.

```
charter validate [-format text|json] [-conformance DIR -profile core-model] [-spec-version V] <dir>...
charter conformance [-format text|json] [-conformance DIR] [-subject reference|none]
charter claim -namespace NS -id ID -implementation ID -implementation-version V [-conformance DIR] [-profile P] [-subject reference|none] [-date YYYY-MM-DD]
charter version [-format text|json] [-conformance DIR]
```

- `validate` loads every `*.json` document below the given directories, checks that each targets the validated specification version, validates it against the embedded schema for its kind, and, when the documents are schema-valid, runs the semantic rules: every implemented rule by default, or the active rules of a profile when `-conformance` names a manifest. Every finding carries its rule id, verification class, document, path, and the normative requirement ids it verifies.
- `conformance` executes every fixture referenced by the active rules of `manifest.yaml`, checking that valid fixtures pass their rules and invalid fixtures fail exactly the rules they declare. Runtime-behavioral fixtures drive their scenario against the subject under test: `-subject reference` (default) is the SDK's own runtime and adapter, `-subject none` leaves behavioral rules untested. Another runtime or adapter runs the same manifest by implementing `validate.Subject` or `validate.AdapterSubject` in Go.
- `claim` runs the manifest and prints a schema-valid `ConformanceClaim` document for one active profile; the claim is `partial` when a profile rule could not be tested and `nonconforming` when a fixture failed.
- `version` prints the module version, the specification and schema-catalog versions the build validates against, and, with `-conformance`, the manifest's specification version and active profiles and rules.

Exit status: `0` conforming, `1` findings or failing fixtures (or a non-conforming claim), `2` usage error or failure to run. Run from the repository root, or pass `-conformance` explicitly.
