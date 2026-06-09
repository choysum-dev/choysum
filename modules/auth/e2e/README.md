# Auth Module E2E Tests

This directory contains system-level end-to-end tests for the `auth` module, orchestrated by `choysum e2e auth`.

## Running

```bash
# Run default scenario (smoke test)
go run . e2e auth

# Run with demo data
go run . e2e auth --with-demo

# Keep temp environment for debugging
go run . e2e auth --keep

# Pass Playwright options
go run . e2e auth -- --headed --project=chromium
```

## Fixtures

- `fixtures/smoke.json`: Creates an admin user (`e2e-admin` / `e2e-admin`) for smoke testing login flows.

## Specs

- `smoke.spec.ts`: Basic login and navigation test to verify auth module functionality.
