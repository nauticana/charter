# Harbor Instance Documents

JSON instances of the Harbor scenario, one document per file named by its `id`, grouped by schema domain. Each validates against the schema for its `kind` in `schema/catalog.json`, and every reference resolves to another document here except `SalesOrder` and `WorkContext` targets explicitly marked with `external: true`.

They illustrate the 1.0.0 schemas and serve as the valid fixture of every semantic rule.
