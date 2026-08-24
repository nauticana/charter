# Binding Package

Provider-neutral interfaces for enterprise systems, system profiles, capability/data/authority/event bindings, feature support, and binding conformance.

Concrete SAP, Salesforce, Oracle, or other vendor implementations belong in downstream modules. Every exported implementation type should include a compile-time assertion for the interface it implements.
