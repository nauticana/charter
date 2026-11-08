# 0009: Work vocabulary and structured sources

Status: Accepted
Date: 2026-11-07

## Context

Consumers matching work found in descriptions and processes to reusable procedures kept their own concept and synonym tables for terms of occupational and skill taxonomies, and two consumers could not exchange a mapping. Evidence records resting on a version of an external artifact encoded its locator, version, and digest into a free-text source by each consumer's own convention.

## Decision

- A classification and its concepts are two kinds. The classification carries publisher, version, location, and license once; a concept carries only its identifier, labels, and links, so uniqueness of an identifier has a scope and a classification version is one document. Neither is enterprise-scoped, like bindings: a taxonomy is a shared catalog.
- Responsibilities, tasks, and capability contracts cite concepts in `conceptIds`. A citation describes the work; it is not an assignment, a role, or a grant, and never a path to authority.
- Charter carries the citation and the cited term, not a classification's content. Redistribution rights stay with the importing product, which the license field makes visible.
- A source stays a `sources` entry: a string, or an object with locator, media type, version, and the digest of the artifact's bytes in the common `digest` form. Verification is the digest comparison a consumer already performs for `derivedFrom`; the schema enforces both the derived-conclusion source requirement and digest well-formedness, so no semantic rule was added.

## Consequences

Downstreams replace concept tables with concept documents and citations, and write artifact provenance in one parseable form. Rule CHR-RULE-ENT-008 keeps identifiers distinct and links within a classification; the Harbor example cites two O*NET-SOC occupations and rests one derived conclusion on a versioned document.

## Specification impact

CHR-ENT-014 and CHR-EVID-017; the `work` schema domain and the `source` definition in the common schema.
