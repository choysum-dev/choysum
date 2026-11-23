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

## Motion / continuity

Page and view busy overlays are **feedback**, not page transitions. Product rules live in
`.dev/docs/web/choy-motion-continuity.md`. Short contract:

1. **Delayed visual busy** — `block` / `skeleton` / page mask / navigation handoff show after
   `loadingTiming.SURFACE_BUSY_DELAY_MS` (= `BAR_DELAY_MS`). Faster work never flashes a mask.
   `aria-busy` / `inert` stay immediate.
2. **Soft stale** — host uses CSS `transition-opacity` (motion tokens); no hard opacity step.
3. **Overlay fade** — `<Transition name="busy-overlay">` (fast duration).
4. **No default full-page route fade** — App shell does not wrap `router-view` in enter/leave
   transitions. Collection routes KeepAlive via `routeShouldKeepAlive(meta)`:
   `meta.keepAlive` (explicit), else `defineRoute` defaults `keepAlive: true` for collection
   route ids (`*_list` / `*_kanban` / `*_chart`; other browse surfaces set keepAlive
   explicitly), else `pageKind: 'collection'`. Opt out with `keepAlive: false`.
   `viewPaths` is ViewSwitcher-only
   and does not imply KeepAlive. Cached instances live only within the current leaf menu
   (`activeMenuId`, sticky across brief null) plus company scope (`keepAliveScopeKey` on
   App shell). Leaving a leaf or switching company remounts the KeepAlive table. Guest shell
   does not use KeepAlive. Cached views reapply `defaultGroups` on `onActivated`.

## Navigation chrome settle (`meta.navLoading`)

Route destinations declare how long the top bar nav ticket stays after navigation.
This is **orthogonal** to `runWithLoading` layers (`surface` / `control` / `chrome`).

| `meta.navLoading` | Chrome held until | Typical destinations |
| --- | --- | --- |
| **`content`** (omit = default) | First surface owns chrome, or idle grace (`NAV_IDLE_SETTLE_MS`, 2s) | App List/Form |
| **`route`** | Route lands (`afterEach`); `settleNavigationIfActive` immediately | Login / Setup / Logout / Register / Error |
| **`none`** | Never opens a nav ticket; drops any leftover nav chrome on success | Silent redirects (opt-in; rarely used) |

Resolver: `modules/web/web/router/navLoadingPolicy.ts` (`resolveNavLoadingPolicy`).
Router hooks in `modules/web/web/router/index.ts`: `content` uses
`scheduleNavigationSettleIfIdle(NAV_IDLE_SETTLE_MS)`; `route`/`none` settle directly.
Timing constants live on `loadingTiming` in the registry — not on RouteMeta
(no per-route millisecond knobs). `NAV_ROUTE_SETTLE_MS` (0) documents the route
policy intent; the router does not schedule a zero-delay idle re-arm.
`SURFACE_BUSY_DELAY_MS` gates page/view/handoff overlay paint (same value as `BAR_DELAY_MS`).

## App create + KeepAlive

- Form **create** (`beginCreate`) runs `DefaultGet` under `layer: 'surface'` so navigation
  chrome hands off instead of sitting for `NAV_IDLE_SETTLE_MS` (create never called `beginDisplay`).
- KeepAlive reactivation can paint without a new surface. `ChoyPage` calls
  `useSettleNavChromeOnActivate` to settle leftover nav chrome on activate when idle.
- Instance cache ≠ data freshness: ModelStore soft-refresh / TTL (when added) stays separate
  from KeepAlive lifetime.

## Registry pointers

- `modules/core/web/client/loadingRegistry.ts` — bar, userBusy, nav ticket, `runWithLoading`
- Setup finish: `submitting` + `runWithLoading({ silent: true })` then `beginNavigationLoading` before `router.replace`
- Notification inbox (`SearchInbox` / mark-read / tip-driven refresh / poll fallback) must use
  `runWithLoading({ silent: true })` so background unread sync never owns the global bar or
  slow-hint (bell UI keeps its local `loading` ref).
