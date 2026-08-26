# Binding Package

`Provider` resolves enterprise systems, system profiles, capability, data, authority, and event bindings, and binding conformance (`BaseProvider` over any `corpus.Source`); `Lookup` finds the binding realizing a capability for a profile. `Features` evaluates declared feature support fail-closed, and `Compare`/`Conformance` evaluate observed support against declarations. `Executor` is the transport contract; `AbstractBinding` enforces feature support around an abstract `VendorMapping` and distinguishes business errors, proven non-execution, and unknown outcomes.

Concrete SAP, Salesforce, Oracle, or other vendor implementations belong in downstream modules. Every exported implementation type includes a compile-time assertion for the interface it implements.
