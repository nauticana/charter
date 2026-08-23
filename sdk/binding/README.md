# Binding Package

The future `binding` package will expose provider-neutral interfaces for system profiles, capability invocation, authority enforcement, event delivery, and evidence collection.

Concrete SAP, Salesforce, Oracle, or other vendor implementations belong in downstream modules. Every exported implementation type should include a compile-time assertion for the interface it implements.
