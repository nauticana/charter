# Conformance Schemas

The draft `conformance_claim` schema records bounded claims for core-model, agent-runtime, and system-adapter profiles. A claim identifies the specification and profile, implementation version, exact tested rules, verification types, results, and test date. Its claim-level fields use the shared `conformanceClaimEnvelope` in `schema/common.schema.json`; `results` adds the general claim's per-rule detail.

This schema does not activate a profile or rule. Activation remains controlled by `conformance/manifest.yaml`.
