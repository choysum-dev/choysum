<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

# Choysum loading feedback

Two channels, one circular glyph family.

## Channels

| Channel | UI | When |
| --- | --- | --- |
| **Chrome** | Top NProgress bar (`--primary`) | Navigation ticket and/or surface that owns the bar |
| **Local** | Lucide `Loader2` via `ChoySpinner` + `text-primary` | Page mask, view `block`, navigation handoff, slow-hint |

Do **not** introduce a third spinner (e.g. thick CSS border discs) in the SPA. Cold boot in `index.html` uses an open SVG arc that matches Loader2 (primary stroke, gap, ~24px).

### Size ladder

- Page / view block / handoff: `size-6`
- Slow-hint corner: `size-3.5`

## Same-wait rules

1. **One primary signal** for a given wait. Setup submit uses page mask + silent RPCs (no chrome bar/chef during Complete→Login).
2. **Handoff** (`ChoyNavigationHandoff` in App shell): while a nav ticket is open and no surface has taken over. Yields when surface starts.
3. **Slow-hint** (corner chef): only when registry userBusy has been pending ≥3s **and** neither handoff nor surface busy is showing.
4. **Empty vs loading**: collection views must not show “No data” until `hasAppliedOnce` and not busy.

## Surface modes (`ChoyViewBusyHost`)

- `block` — Loader2 overlay (first frame with no rows)
- `stale` — keep content, light fade (refresh with prior rows)
- `skeleton` — reserved; prefer `block` for first paint so local feedback stays Loader2-only

## Registry pointers

- `modules/core/web/client/loadingRegistry.ts` — bar, userBusy, nav ticket, `runWithLoading`
- Setup finish: `submitting` + `runWithLoading({ silent: true })` then `beginNavigationLoading` before `router.replace`
