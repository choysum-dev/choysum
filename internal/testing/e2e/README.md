This package implements Go-side orchestration for system E2E.

## Engines (migration)

Specs are routed **per file**:

- Import `@playwright/test` → Playwright (legacy Node path)
- Otherwise → QuickJS + `@choysum/e2e` + chromedp (`runE2EHost`)

Within a scenario, QJS specs run first, then Playwright specs (shared `runtime.json` / server).

## Chromium (QJS / chromedp)

Install Chrome for Testing (non-npm):

```bash
python3 scripts/ci/install_chromium.py
# or
./choysum test e2e --install-browser
```

Binary resolution order:

1. `CHOYSUM_CHROMIUM_PATH` (absolute path to the chrome binary)
2. `$CHOYSUM_HOME/browsers/chromium-<rev>/…` (default `CHOYSUM_HOME=~/.choysum`)
3. Local system Chrome/Chromium (local only)

Headless is the default; set `CHOYSUM_E2E_HEADED=1` for headed runs.

## Playwright note

E2E scenarios use a single sqlite file DB per run. To avoid flaky `database is locked` errors from concurrent writes (e.g. login token creation), the generated Playwright config defaults to `workers=1`. You can override it via `-- --workers=N`.

Design source of truth: docs/e2e_by_module.md
