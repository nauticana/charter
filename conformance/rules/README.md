# Conformance Rules

Machine-checkable semantic and behavioral rules live here. Each rule must have a stable identifier of the form `CHR-RULE-<AREA>-<NUMBER>`, cite one or more normative `CHR-*` requirement IDs, state its applicable profile, and define an unambiguous pass or fail result.

For example, rule `CHR-RULE-AUTH-001` might verify requirement `CHR-AUTH-002`. The visibly distinct namespaces allow reports and manifests to distinguish verification procedures from the requirements they verify.

Do not duplicate constraints already enforced completely by JSON Schema; reference the applicable schema instead.
