# Derivation Package

`Deriver` adopts documents of a source namespace into a target namespace (CHR-CONF-015). `Derive` reads each `Root` at its version label from a `corpus.VersionSource`, follows the schema's idRef and objectRef properties to every document of the source namespace they reach at its current version, and returns, ordered by identifier, the copies to save. A copy:

- lives in the target namespace and belongs to the target enterprise when its kind is enterprise-scoped;
- names its explicitly namespaced references to the source namespace in the target namespace, where their copies live, and keeps references to any other namespace;
- drops the evidence and external identifiers of the source set, keeping evidence cited in another namespace;
- records in `derivedFrom` the source, its version (the label its kind declares, otherwise `Options.SourceVersion`), its `corpus.ContentDigest`, and `adopted`.

A target document already derived from the same version and content is reused. One derived from another version or content is reused, or with `Options.Upgrade` replaced, unless it was tailored (`ErrTailored`) or its derivation is fixed because the replaced version is published (`ErrDerivationFixed`); a new version of a versioned definition records its own derivation. A target document of a copied identifier not derived from that source is refused (`ErrForeignDocument`). Enterprises and records of what happened in the source set (actions, evidence, exceptions, escalations, approvals, mediation decisions, process and task instances, binding conformance, conformance claims) are never copied (`ErrNotDerivable`).

The deriver writes nothing; the caller saves the copies, which a store validates as any other write.
