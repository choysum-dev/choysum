# Validation Subdomain Boundaries

`validation/` owns the repository-side validation bridge and validation issue explanation logic for write paths.

Public entry point:

- Expose this subdomain to `repository.ts` and other write-path assemblers only through `./index.ts`.

Content that belongs in this subdomain:

- repository-side bridging for the runtime validation facade
- validation issue and error wrapping
- mapping SQL constraint errors into validation issues
- validation policy helpers such as platform whitelist, rejectUnknown, and audit bucket handling

Content that does not belong in this subdomain:

- read-path query helpers
- payload encode or decode, selection or projection, and authz or record-rule coordination
- generic validation logic that belongs to the runtime layer; that logic should stay in `runtime/validation`

Maintenance rules:

- When adding repository validation helpers, place them in `validation/` instead of adding new root-level `repository_validation_*.ts` files.
- Call sites across the repo should prefer the `./validation` barrel; direct relative imports of concrete files are only allowed between implementations inside this subdomain.
- If a helper needs request or runtime state, first check whether it is still a repository bridge; if it has evolved into generic runtime logic, move it back into `runtime/validation`.