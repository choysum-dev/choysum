This package implements Go-side orchestration for system E2E.

## Engine

All module E2E specs run on **QuickJS + `@choysum/e2e` + chromedp** (`runE2EHost`).
There is no Playwright / Node test runner path.

## Chromium

Install Chrome for Testing (non-npm):

```bash
python3 scripts/ci/install_chromium.py
# or
./choysum test e2e --install-browser
```

Binary resolution order:

1. `CHOYSUM_CHROMIUM_PATH` (absolute path to the chrome binary)
2. `$CHOYSUM_HOME/browsers/chromium-<rev>/…` (default `CHOYSUM_HOME=~/.choysum`)
3. Local system Chrome/Chromium (**local only**; CI never falls back)

Headless is the default (unset or `CHOYSUM_E2E_HEADED=0`). Set
`CHOYSUM_E2E_HEADED=1` only for local debugging with a visible browser; do not
set it in CI.

## Spec filters

Pass path/name filters after `--` (for example `choysum test e2e auth -- smoke.spec.ts`).
Flag-looking args (leading `-`) are ignored; use `CHOYSUM_E2E_HEADED=1` for a visible browser.
Workers are fixed at 1 (shared sqlite DB per scenario).

## Illegal imports

Specs must not import `@playwright/test` or Node builtins `node:fs` /
`node:path` / `node:crypto`. The runner fails fast when those appear.
