# 0008: JSON-LD interchange

Status: Accepted
Date: 2026-11-07

## Context

A customer with an existing semantic layer may need to exchange its enterprise model with Charter rather than remodel it. Charter stores JSON documents validated by JSON Schemas; RDF tooling expects a vocabulary, stable IRIs, and a context.

## Decision

- The exchange format is JSON-LD 1.1 in the compact form the SDK writes, under one published context generated from the schemas (`schema/context.jsonld`, `charter context`). Turtle and other RDF serializations are produced by the customer's JSON-LD processor; Charter ships no RDF parser.
- Every Charter property name is a vocabulary term under `@vocab`; kinds are the `@type` values. The context is regenerated with the schemas, and a test fails when the published file falls behind.
- A document's IRI is a URN of its namespace and identifier, so identity does not depend on where the set is hosted and a URI namespace is carried unchanged by percent-encoding. Objects outside the set keep their kind in an external URN; same-namespace references compact back to bare identifiers.
- Shape properties and relationships are exported as `rdf:Property` nodes of the definition's IRI, so a semantic layer can state facts in Charter's vocabulary. They are derived from the shape and are skipped on import.
- Import is total or refused: content Charter cannot carry is listed as loss and the import fails unless the caller passes exactly that list back, which is the record of accepted loss. Properties of another vocabulary are not loss; they are kept as namespaced extensions.

## Consequences

`charter export` and `charter import` carry a document set across without a downstream mapping. The importer reads the compact form only; a graph in another JSON-LD form is compacted with a standard processor against the published context first.

## Specification impact

CHR-CONF-016.
