# Harbor Instance Documents

JSON instances of the Harbor scenario, one document per file named by its `id`, grouped by schema domain. Each validates against the schema for its `kind` in `schema/catalog.json`, and every reference resolves to another document here except external `SalesOrder` and `WorkContext` targets.

They illustrate the draft schemas; they are not conformance fixtures.
