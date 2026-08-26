# Corpus Package

`Corpus` indexes documents by namespace and id and resolves same-namespace or explicitly namespaced references. `Loader` is the interface, `BaseFSLoader` the file-system implementation; `Parser` builds documents from JSON; `Decode[T]` produces typed documents. `Source` is the abstract loading contract behind every provider (a `Corpus` is one), and `AbstractDocumentProvider` adds kind validation, reference resolution, and effective-time queries over it; `ResolveAs`, `ListAs`, and `EffectiveAs` return typed documents.
