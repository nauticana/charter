# Go Reference SDK

Status: Planned; no SDK packages have been implemented.

The Go reference SDK will demonstrate the Charter schemas and conformance behavior without becoming the source of normative semantics. Independent implementations must be able to conform without importing this SDK.

Planned packages:

- `model`: Go representations generated from or checked against released schemas
- `validate`: structural and semantic validation using the shared conformance corpus
- `binding`: exported neutral interfaces for downstream adapter implementations

The repository root currently owns the Go module. If the SDK later requires an independent release cadence, extraction and import-path compatibility will be planned as a versioned migration rather than assumed to be a directory-only move.
