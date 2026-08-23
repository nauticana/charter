# Contributing to Nauticana Charter

Nauticana Charter welcomes issues, design proposals, specification changes, schemas, conformance cases, examples, documentation, and reference implementation work.

## Before contributing

Read the specification overview, governance policy, and code of conduct before opening a substantial change. For a breaking or cross-cutting change, open an architecture decision record or design issue before implementation.

Contributions must be original work that the contributor has the right to submit under the Apache License 2.0. Do not submit confidential customer information, proprietary vendor content, credentials, or material copied from licensed reference documentation.

## Change categories

- Editorial changes clarify text without changing observable requirements.
- Compatible changes add optional behavior without invalidating conforming implementations.
- Breaking changes alter normative requirements, schemas, identifiers, or conformance outcomes.
- Supporting changes affect examples, tools, or SDK code without changing the specification contract.

Pull requests that change `spec/`, `schema/`, or `conformance/` must identify the affected requirement IDs and state whether the change is editorial, compatible, or breaking.

## Normative changes

A normative change should update all affected artifacts in the same pull request:

1. Specification prose and requirement identifiers
2. Machine-readable schemas
3. Conformance rules and manifest entries
4. Valid and invalid fixtures
5. Examples and reference SDK behavior, when applicable
6. Migration or compatibility guidance for breaking changes

Normative prose and schemas are jointly authoritative. A disagreement between them is a defect; do not resolve it silently in favor of one artifact.

## Pull requests

Keep changes focused and explain the problem, proposed behavior, compatibility effect, and verification performed. Tests and fixtures should demonstrate both accepted and rejected behavior where practical.

By contributing, you agree to follow the project's code of conduct and governance process.
