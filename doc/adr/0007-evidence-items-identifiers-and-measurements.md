# 0007: Structured evidence items, external identifiers, and measurements

Status: Accepted
Date: 2026-11-06

## Context

Consumers that synchronize an enterprise from an HR or ERP system, compile the facts a decision rests on, or report outcome results kept three kinds of private tables beside Charter: per-item evidence labels on tasks and contracts, the identifiers an object carries in a customer system, and measured outcome values with the actions they rest on. Each was proposed as an upstream gap, together with a fourth: authority documents as ranked fact sources.

## Decision

- An evidence requirement may be a structured item: the information and attribute whose fact the work rests on, whether it is required, and a maximum age no wider than the information's own. Items are facts, never authority. The controlling authority of an action is already structured: the contract's `requiredAuthority` and `constraints`, and the record's authority, approval, and separation-of-duties evaluations (CHR-EVID-002, CHR-EVID-010).
- Authority documents are not fact sources, so `factAuthority.ranks` keeps naming system profiles and evidence categories. A value a person decided enters as a `human-decision` record whose `sources` cite the approval, and that category is already rankable. Making a grant or approval a fact source would evaluate authority twice, once by the authority rules and once by fact standing.
- A mutating contract observes its required facts before the operation is submitted; a read relies on the facts it returns. Observing before a read would observe the read.
- External identifiers are correlation keys on the organization and human identity documents an external system holds, carried on the envelope and admitted by those schemas only, one effective per system profile at a time. They are not checked against an enterprise because system profiles are not enterprise-scoped.
- A measurement is stated by an evidence record, not by the outcome measure: results accumulate as evidence and are found by querying the measure (`evidence.Queries.Measurements`). The baseline keeps its value and window because the CHR-PROC-009 gate reads them from the measure, and a baseline citing a measurement record must agree with it.

## Consequences

Downstreams drop the private tables and keep only their own provenance of each item. Rules CHR-RULE-CAP-003, CHR-RULE-ENT-007, CHR-RULE-PROC-006, and CHR-RULE-RT-014 enforce the semantics; the Harbor example carries one instance of each.

## Specification impact

CHR-CAP-013, CHR-ENT-013, and CHR-PROC-011; `evidenceRequirement` and `externalIdentifiers` in the common schema and `measurement` on `EvidenceRecord`.
