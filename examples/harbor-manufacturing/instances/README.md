# Harbor Instance Documents

JSON instances of the Harbor scenario, one document per file named by its `id`, grouped by schema domain. Each validates against the schema for its `kind` in `schema/catalog.json`, and every reference resolves to another document here except targets of kinds outside the catalogue, such as `SalesOrder`, explicitly marked with `external: true`; a `derivedFrom` source is never resolved here.

They illustrate the current schemas and serve as the valid fixture of every semantic rule.
