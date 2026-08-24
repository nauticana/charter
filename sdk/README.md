# Go Reference SDK

Status: Planned; no SDK packages have been implemented.

The Go reference SDK builds on Keel as the backend framework; Scout, the agent runtime and MCP server, implements its agent contracts. The specification, schemas, and conformance corpus require no platform, and independent implementations must be able to conform without importing this SDK. Keel never depends on Charter, and Charter never depends on Scout.

Planned packages:

- `model`: schema-aligned document and reference types
- `identity`, `organization`, `agent`, `process`, `architecture`: provider and resolution contracts
- `authority`, `capability`, `binding`: governed decision and invocation contracts
- `evidence`, `information`: evidence and information-governance contracts
- `validate`: structural/semantic validation, fixture execution, and conformance claims

`Abstract*` types are embed-only extension points that implement the governed logic and leave storage, transport, or lookup to the downstream; `Base*` types are complete and usable as-is. Both use Keel's ports directly.

The repository root currently owns the Go module. If the SDK later requires an independent release cadence, extraction and import-path compatibility will be planned as a versioned migration rather than assumed to be a directory-only move.
