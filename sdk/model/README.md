# Model Package

Structs only: the `Envelope` base every typed document embeds, `Ref`/`ObjectRef`/`Validity`/`Limit` value types, typed kinds used by the semantic rules, and the generic `Document`. `Ref` carries the one method in the package, its JSON codec, because `encoding/json` requires it on the type. Types for the remaining kinds will be generated from the schemas; generated files stay separate from hand-written behavior and add no fields absent from the normative contract.
