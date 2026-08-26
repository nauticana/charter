# Process SDK

`DefinitionProvider` and `InstanceProvider` keep definitions and execution context distinct (`BaseProvider` implements both). `Graph` reads hierarchy, sequence, and dependency relationships as declared, and `BaseContextResolver` builds a `TaskContext` from a task instance, failing closed on definition mismatch, performer kind, participation, or an assignment not effective at instance time.
