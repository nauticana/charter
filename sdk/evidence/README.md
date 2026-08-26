# Evidence SDK

`Sink` is the append-only output of governed work; `AbstractSink` implements duplicate rejection, supersession links, and hash-chained bundle integrity over an abstract `Store` (`BaseMemoryStore`, `BaseMemorySink`). `Provider` reads records back (`BaseProvider`), `Queries` derives actor attribution and supersession lineage, and `Verifier` recomputes bundle integrity (`BaseSHA256Digester`).
