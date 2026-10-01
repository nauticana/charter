# Binding Schemas

Schemas for the `CHR-BIND` domain: `enterprise_system`, `system_profile`, `capability_binding`, `data_binding`, `authority_binding`, `event_binding`, and `binding_conformance`. `package_manifest` defines reusable vendor realizations and is not a Charter document kind.

Binding documents are not enterprise-scoped; profiles and bindings may be shared catalogs. Every binding identifies its version, target profile, and feature support. `binding_conformance` uses the shared `conformanceClaimEnvelope` in `schema/common.schema.json` and adds binding-specific feature evaluation. Capability bindings map inputs, outputs, business errors, authority checks, idempotency, and evidence. Vendor-specific bindings may appear as non-normative examples but do not enter Charter core semantics.

## Relations

```mermaid
flowchart LR
    BindingConformance -->|bindingId| CapabilityBinding
    CapabilityBinding -->|capabilityId| CapabilityContract
    CapabilityBinding -->|systemProfileId| SystemProfile
    AuthorityBinding -->|capabilityId| CapabilityContract
    AuthorityBinding -->|systemProfileId| SystemProfile
    EventBinding -->|systemProfileId| SystemProfile
    DataBinding -->|systemProfileId| SystemProfile
    DataBinding -->|informationDefinitionId| InformationDefinition
    SystemProfile -->|enterpriseSystemId| EnterpriseSystem

    classDef ext stroke-dasharray: 4 3
    class CapabilityContract,InformationDefinition ext
```

Dashed nodes belong to other domains. `BindingConformance.bindingId` may name any binding kind.
