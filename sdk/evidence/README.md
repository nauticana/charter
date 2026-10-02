# Evidence SDK

`Sink` is the append-only output of governed work; `AbstractSink` implements duplicate rejection, supersession links, and hash-chained bundle integrity over an abstract `Store` (`BaseMemoryStore`, `BaseMemorySink`). `Redactor` (`BaseRedactor`) masks credentials inside evidence before it is stored or published. `Provider` reads records back (`BaseProvider`), `Queries` derives actor attribution, the actions of one execution context (`ActionsIn`, from `Provider.ActionsIn`, which a durable provider answers from an index), supersession lineage, and the records about a subject (`RecordsAbout`), `AbstractSink.Assemble` bundles them with a computed integrity value, and `Verifier` recomputes it (`BaseSHA256Digester`); `Seal` and `Verifier.VerifyRecord` do the same for the content digest of a single record.

`evidencetest.Run` is the contract suite a `Store` and `Provider` implementation runs from its own tests.
