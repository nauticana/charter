# Charter Specification Overview

Status: Draft, unreleased

This directory contains the normative prose for the Nauticana Charter specification. Charter defines a vendor-neutral model for aligning human accountability, governed agents, business processes, authority, and enterprise-system capabilities.

## Scope

Charter specifies enterprise and agent identities, responsibilities, processes, authority, capability contracts, external-system bindings, architecture states, evidence, information governance, security expectations, and conformance. It does not prescribe an agent runtime, model provider, workflow engine, enterprise application, transport protocol, or user interface.

## Normative language

The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **NOT RECOMMENDED**, **MAY**, and **OPTIONAL** are to be interpreted as described in BCP 14, RFC 2119 and RFC 8174, when and only when they appear in uppercase.

Lowercase uses of those words have their ordinary meaning.

## Authority of artifacts

Normative prose in `spec/` and machine-readable schemas in `schema/` are jointly authoritative. Schemas determine structural validity. Normative prose and conformance rules determine semantic and behavioral conformance. A disagreement among these artifacts is a specification defect.

Files under `doc/`, `examples/`, `sdk/`, and `cmd/` are non-normative. They cannot create requirements that do not exist in the normative contract.

## Requirement identifiers

Every independently testable normative requirement receives a stable identifier of the form `CHR-<AREA>-<NUMBER>`. Identifiers MUST NOT be reused for a different requirement. Removed identifiers remain reserved.

- `ENT`: enterprise structure and responsibility
- `ID`: identity and lifecycle
- `AGENT`: agent behavior
- `PROC`: processes and tasks
- `AUTH`: authority and delegation
- `CAP`: capabilities and operations
- `BIND`: external-system bindings
- `ARCH`: architecture states and roadmaps
- `EVID`: evidence and audit
- `INFO`: information governance
- `SEC`: security
- `CONF`: conformance and compatibility

## General conformance

Cross-cutting conformance, extension, validation, and identifier requirements are defined together in [Conformance and Compatibility](12_conformance.md). This overview introduces their conventions but does not define a separate requirement area.

## Versioning

Specification releases use semantic versioning. Git tags identify immutable releases; the repository contains the current development version. Machine-readable artifacts carry an explicit specification version so that copied documents remain identifiable outside the repository.

## References

- [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119)
- [RFC 8174](https://www.rfc-editor.org/rfc/rfc8174)
