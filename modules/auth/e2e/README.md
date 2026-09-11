# Auth Module E2E Tests

This directory contains system-level end-to-end tests for the `auth` module,
orchestrated by `choysum test e2e auth` (QuickJS + chromedp).

## Running

```bash
# Run default scenario (all specs under e2e/)
./choysum test e2e auth

# Run with demo data
./choysum test e2e auth --with-demo

# Keep temp environment for debugging
./choysum test e2e auth --keep

# Filter specs by path/name
./choysum test e2e auth -- smoke.spec.ts
```

Requires a Chrome/Chromium binary on the machine (or set
`CHOYSUM_CHROMIUM_PATH`).

## Fixtures

- `fixtures/smoke.json`: Creates an admin user (`e2e-admin` / `e2e-admin`) for smoke testing login flows.

## Specs

- `smoke.spec.ts`: Basic login and navigation test to verify auth module functionality.
