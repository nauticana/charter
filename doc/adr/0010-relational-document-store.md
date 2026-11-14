# 0010: Relational document store

Status: Accepted
Date: 2026-11-09

## Context

The keel adapter kept each document as a JSON revision in `charter_document_revision` and each evidence document as a row of keel's table change log. Neither table could be queried by property, modeled object references were not foreign keys, and a document set was only as consistent as the validation run before each write. [schema/database.md](../../schema/database.md) designs the relational model that replaces them: one table per kind and per array property, in BCNF, modeled object references as foreign keys, and enumerations as keel code lists.

## Decision

- The tables are keel schema modules under `sdk/schema`, one YAML file per table, embedded by the `sdk/schema` package so the adapter reads the same definitions keel's `schemagen` turns into DDL. The adapter's own directory holds only Go.
- The modules depend only on lower modules, so a downstream installs a subset. A table referencing a higher module than its owner's belongs to that higher module, and each module seeds the code lists it owns. The store is built for the installed modules and refuses kinds and properties outside them.
- Each kind is mapped by a declarative specification: which table holds it, which column each property occupies, and which child table each list becomes. One engine decomposes a document into rows and composes it back, so every kind follows the same rules for references, versions, codes, and lists. A property no column holds is refused rather than dropped.
- Named SQL is generated once per table from the embedded definitions: insert, update by key, and select and delete by the owning column. NUMERIC travels as text both ways, so no value passes through a float.
- `DocumentStore` saves governance documents and appends evidence into the same tables. A versioned definition stores each declared version once; a mutable document is updated in place and its child rows rewritten; a mediation decision and every evidence kind are written once. A save is a replay when its rows equal the stored rows.
- Facts the model derives instead of storing are checked on write: a runtime's identity is its definition's, an agent acting through a runtime is that runtime's identity, a stated currency exponent is keel's, and a pinned version label names a stored version.
- `approvedAction` is stored as a capability contract when the string names one stored in the document's namespace, and as an action class otherwise, until the schema separates the two.
- A reference to an object outside the document set registers that object once, so it has a key like any document.
- A downstream table references a stored document by its `enterprise_object` id. `DocumentStore.ObjectIDs` and `ObjectRefs` translate between ids and references in one query for any number of documents, checking each kind, so no downstream queries `enterprise_object` or copies the adapter's SQL. A downstream column that holds one kind still names that kind's table in its foreign key.
- The store's tests run the generated SQL against an in-memory database that enforces the modules' keys, unique indexes, foreign keys, NOT NULL columns, widths, and code-list checks.

## Consequences

`TableLogStore` and its `ChangeLog` requirement are removed; `DocumentStore.Evidence(partnerID)` is the evidence store, and `Source(partnerID)` serves evidence with the documents, so a save's references to evidence resolve without a second source. Writes use named SQL and do not enter keel's table change log; mutable row history is not retained. Reading a document costs one query per table its kind uses, plus one per referenced object set, tree level, and currency set; listing a kind reads each table once for all its documents. Because evidence is appended separately, a document whose optional property cites evidence that names the document, such as a measure whose baseline cites a measurement of it, is saved without that property first and completed once the evidence exists. Stored data in the previous tables is not migrated.

## Specification impact

None. The rules the model adds to saving and the split of `approvedAction` remain open points in the relational model.
