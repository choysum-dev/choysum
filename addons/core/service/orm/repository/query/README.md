# Query Subdomain Boundaries

`query/` owns pure query-shaping logic used while building repository read queries.

Public entry point:

- Expose this subdomain to `repository.ts`, `read/`, `projection/`, and other consumers only through `./index.ts`.

Content that belongs in this subdomain:

- condition compilation and normalization
- having, ordering, and select-context helpers
- query helpers that integrate with ORM metadata without triggering write-path side effects

Content that does not belong in this subdomain:

- write-path validation, payload handling, and authz or field-rule or record-rule coordination
- runtime request reads, caching, or side-effecting bridges
- temporary helpers that only serve a single orchestrator; colocate those with the owning subdomain instead

Maintenance rules:

- When adding a new helper, first decide whether it belongs in `query/` instead of letting it drift back into the repository root.
- Files may be split further inside this subdomain, but imports across the repo should prefer the `./query` barrel over direct internal file paths.
- If logic is shared across multiple read-side subdomains such as projection or read, keep it in `query/`; if it starts carrying relation projection or codec semantics, move it to `projection/`.