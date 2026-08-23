# Charter Schemas

Status: Planned; no normative schema release exists yet.

This directory will contain language-independent JSON Schemas for Charter instance documents. Schema structure mirrors the specification domains:

- `enterprise/`: enterprises, organization-unit trees, positions, roles, responsibilities, and assignments
- `agent/`: identities, agents, lifecycle, authority, approval, and escalation
- `process/`: value streams, processes, tasks, capabilities, and architecture states
- `binding/`: system profiles and capability, data, authority, and event bindings
- `evidence/`: action evidence, decisions, exceptions, audit trails, and information provenance

## Authority and versioning

Schemas and normative prose are jointly authoritative. Schemas determine structural instance validity but do not override behavioral requirements in `spec/`.

Each released schema will include:

- A stable `$id`
- The Charter specification version
- A schema artifact version
- Explicit extension points
- References to the normative requirement IDs it structurally enforces

Schemas will be added only with matching valid and invalid conformance fixtures. Draft placeholder schemas that accept arbitrary content are intentionally avoided.
