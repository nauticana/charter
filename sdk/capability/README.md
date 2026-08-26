# Capability SDK

`Catalog` serves capability contracts (`BaseCatalog`); `VersionPolicy` decides contract compatibility. `AbstractInvoker` runs the governed pipeline around an abstract `binding.Executor`: version, authority, approval, separation of duties (`SodChecker`), binding feature support, idempotency (`Ledger`), execution, declared-outcome and business-error verification, and an `ActionRecord` appended to the evidence sink for every decision. Tool and system adapters implement the transport; they never replace the contract.
