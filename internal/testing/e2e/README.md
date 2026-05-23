This package implements Go-side orchestration for system E2E.

Note: E2E scenarios use a single sqlite file DB per run. To avoid flaky `database is locked` errors from concurrent writes (e.g. login token creation), the generated Playwright config defaults to `workers=1`. You can override it via `-- --workers=N`.

Design source of truth: docs/e2e_by_module.md
