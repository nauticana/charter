# 0001: Enterprise object-relationship model

Status: Accepted
Date: 2026-08-23

## Context

Mature HR systems (SAP HRP1000/HRP1001) model organization objects and typed, effective-dated relationships as separate records. Simpler SAP-inspired relational models use a single-parent organization tree with positions instantiated per unit and time-dependent assignments. The Charter enterprise schemas must represent both without vendor fields.

## Decision

Definitional structure is a field; everything else is a record. `OrganizationUnit.parentUnitId` carries the single rooted tree (CHR-ENT-008) and `Position.organizationUnitId` carries placement (CHR-ENT-009). All other relations are effective-dated documents: `Assignment` for identity-to-work connections, `OrganizationRelationship` for typed relations (`reports-to`, `deputy-of`, namespaced custom types). Positions have standalone stable identifiers, independent of unit and type.

## Consequences

A relational model maps one-to-one (parent column → field, assignment row → document); an object-relationship catalog maps its relationship records to `OrganizationRelationship` without schema changes. Structural cycles and cross-document referential integrity are semantic rules for conformance, not schema checks.

## Specification impact

None; implements `spec/01_enterprise.md` as released. Schemas annotate their requirements via `x-charter-requirements`.
