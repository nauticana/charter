# Governance

Nauticana Charter is an independently maintained open-source project. It is not governed by or affiliated with the Apache Software Foundation.

## Decision making

The project uses maintainer-led, issue-informed decision making. Maintainers are responsible for releases, repository access, security response, specification consistency, and final acceptance of changes. Material decisions should be recorded in public issues, pull requests, or architecture decision records unless confidentiality is required for security reasons.

Consensus is preferred. When consensus cannot be reached, maintainers decide based on the vendor-neutral purpose of Charter, compatibility, implementability, security, and the needs of independent downstream implementers.

## Specification authority

The following paths form the versioned contract:

- `spec/` for normative semantics and behavioral requirements
- `schema/` for normative instance structure and structural constraints
- `conformance/` for conformance profiles, rules, traceability, and fixtures

Examples, SDKs, tools, and explanatory documents are non-normative. They must remain consistent with the versioned contract but cannot create new normative requirements.

## Releases and compatibility

Specification releases use semantic versioning:

- A major release may contain breaking normative changes.
- A minor release may add backward-compatible requirements or optional features.
- A patch release may contain editorial corrections and defect fixes that do not intentionally alter conforming behavior.

Every release must identify its specification version, publish its compatibility impact, validate all conformance fixtures, and state the supported schema and SDK versions.

## Maintainers

Maintainers are contributors entrusted with review and repository responsibilities through sustained, constructive participation. Existing maintainers appoint or remove maintainers through a documented decision. Repository permissions are operational authority, not ownership of contributed work.

## Amendments

Governance changes require a pull request that explains the motivation and transition impact. They are not specification changes unless they also alter files in the versioned contract.
