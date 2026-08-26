# Agent SDK

`Provider` resolves agent identities, versioned definitions, and runtime instances (`BaseProvider` over any `corpus.Source`). `ExecutionContext` attributes work to the identity, runtime, definition version, and single assignment in whose context authority is evaluated; `BaseAdmission` decides, fail-closed and citing the requirement it enforces, whether a runtime may accept that work. Scout is the reference runtime implementation; downstream runtimes implement the same interfaces.
