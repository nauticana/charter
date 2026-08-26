# Model Package

Structs only: the `Envelope` base every typed document embeds, `Ref`/`ObjectRef`/`Validity`/`Limit` value types, the `Kind` constants, one hand-maintained struct per catalogued kind grouped by schema domain, and the generic `Document`. `Ref` carries the one method in the package, its JSON codec, because `encoding/json` requires it on the type. The test round-trips every Harbor document through its struct and back through its schema, so the structs add no fields absent from the normative contract and drop none.
