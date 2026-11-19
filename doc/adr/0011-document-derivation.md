# 0011: Document derivation

Status: Accepted
Date: 2026-11-16

## Context

CHR-CONF-015 requires a document derived from another document set to record the source, its version, and its content digest, but no operation produced such a document. Every consumer adopting reference documents into its own namespace repeated the same steps: read the pinned sources, follow their references, relocate the copies, record the derivation, and reuse or refuse what the target already holds. The content digest had no defined form, so two implementations could not agree on it.

## Decision

- `derivation.Deriver` returns the copies to save and writes nothing; a store validates them as any other write.
- The content digest is the `sha256` digest of the document's JSON canonicalized as RFC 8785 specifies (`corpus.ContentDigest`), so it does not depend on how the document was stored or serialized.
- A root is read at its version label through `corpus.Source.FetchVersion`, which every source implements; documents reached through references are read at their current version. A source whose kind declares no version records the version the caller names, such as the source set's release.
- References follow the schema's idRef and objectRef properties (`validate.SchemaMeta.WalkRefs`). A reference to the source namespace names the target, where its copy lives; the enterprise is the target's; evidence and external identifiers of the source set are dropped.
- Enterprises, human identities, agent runtimes, and records of what happened are never copied: the people, runtimes, and records of the source set are not the target's.
- A copy already derived from the same version and content is reused; another is kept, the references of the newer source not followed, and replaced only on request, never when tailored or when its derivation is fixed.
- A derivation is fixed per version: a new version of a versioned definition records its own derivation, as CHR-AGENT-010 states, and `validate.Additions` compares the derivation of a published document only when the write replaces that version.

## Consequences

Consumers adopt reference documents with one call and record digests a verifier can recompute. Upgrading an adopted versioned definition writes a new version with its own derivation.

## Specification impact

None. CHR-CONF-015 reads, for a versioned definition, the document as its version, as CHR-AGENT-010 states.
