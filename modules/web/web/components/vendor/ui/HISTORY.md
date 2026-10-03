<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

# vendor/ui product patches

Hand-adapted shadcn-vue L2 under this directory. Upstream is not synced by CLI;
product contract patches are recorded here so a future re-copy can replay them
after merging Reka/shadcn markup.

## Patch policy

**Allowed (must stay after an upstream refresh):**

- Bind default control height to `--choy-control-height` via theme utilities
  (`h-control`, `h-control-sm`, `h-control-lg`, `size-control`) — not hard-coded
  `h-9` / `h-10` / `h-8` on default Button / Input / Select trigger / Combobox
  input / TabsList (and icon Button → `size-control`).
- Logical inset/padding for directional chrome (`ps` / `pe` / `start` / `end`
  instead of physical `pl` / `pr` / `left` / `right` where the control mirrors).
- Focus rings: prefer `focus-visible:ring-*` on interactive L2 hosts.
- Token / theme color aliases (`bg-primary`, `ring-ring`, …) already mapped in
  `modules/web/web/styles/theme.css`.

**Not allowed here (put in Choy* L1 or pages instead):**

- Product layout, Field chrome, table hosts, shell chrome.
- One-off visual polish unrelated to the control ruler or a11y/RTL defaults.

**Replay after refresh:** re-apply every entry below that still applies, then
update `VENDOR.json` (`rekaVersion` / `lucideVersion` / `copiedAt`) and append a
new HISTORY entry describing what was re-copied vs re-patched.

---

## Entries (newest first)

### 2026-10-03 — W1 shell hard-cut FE SFC / Sheet adaptations

- **Epic:** shadcn-vue alignment W1 (`ChoyWebShell` → Sidebar*).
- **QuickJS FE:** avoid `defineProps<X>` where `X` is imported from `reka-ui`
  (compiler rejects non-relative type imports). Inlined local props on Sidebar
  Primitive hosts, Collapsible, Sheet, Avatar Fallback/Image.
- **Sheet:** `Sheet.vue` uses `defineModel('open')` like product Dialog; Content /
  Title / Description / Overlay drop `Dialog*Props` type imports; logical
  `start`/`end` / `border-s`/`border-e` on left/right sheets.
- **Sidebar:** `Sidebar.vue` no longer imports types from `"."` (directory); Sheet
  children imported as concrete `.vue` files; cookie defaultOpen null-safe;
  `sidebarMenuButtonVariants` moved to `sidebarMenuButtonVariants.ts`.
- **Do not regress:** density `h-control*` / `size-control` on menu button variants.

### 2026-10-03 — W0 full L2 vendoring + semantic token pairs

- **Epic:** shadcn-vue alignment W0 (`choy-shadcn-vue-alignment.md`).
- **Upstream:** `unovue/shadcn-vue@67c9a392` `apps/v4/registry/new-york-v4/ui` via
  `scripts/web/vendor_shadcn_ui_w0.py` (missing families only; existing 19 kept).
- **Tokens / theme (outside this folder, required together):**
  - `tokens.css`: add `primary-foreground`, `card`/`popover`/`secondary`/`accent`/
    `input`, full `sidebar-*`, `--choy-radius` base; short shadcn aliases
    (`--popover`, `--primary`, …) for upstream `var(--*)` usage.
  - `theme.css`: map new `--color-*` utilities (incl. `sidebar-*`, `chart-*`).
- **Adaptations on new L2:**
  - `@/lib/utils` → relative `lib/utils`; `@lucide/vue` → `lucide-vue-next`
    (strip `Icon` suffix); `@/registry/.../ui/X` → relative sibling imports.
  - `class-variance-authority` → local `lib/cva.ts`.
  - Default heights `h-9`/`h-10`/`h-8`/`size-9` → `h-control*` / `size-control`;
    `shadow-xs` → `shadow-sm`; `pl`/`pr` → `ps`/`pe`.
  - Kept `button/index.ts` exports `buttonVariants` for AlertDialog/Calendar/
    Pagination composers; Button.vue / Badge.vue recipes use `*-foreground` pairs.
- **Peers added on `@choysum-dev/web` for niche L2:** `vue-sonner`,
  `embla-carousel-vue`, `vue-input-otp`, `vee-validate`.
- **Do not regress:** density ruler + domain ban on `vendor/ui` imports.

### 2026-09-30 — control-height ruler + logical chrome

- **Commit:** `a3addc60` (`feat(web): bind L2 controls to density control-height tokens`).
- **Tokens / theme (outside this folder, required together):**
  - `tokens.css`: `--choy-control-height` 32px comfortable / 28px under
    `html[data-density=compact]`; `--choy-component-size-base` aliases it;
    sm/lg companion sizes.
  - `theme.css`: `--height-control` → `h-control`; `--height-control-sm` /
    `--height-control-lg`; `--size-control` → `size-control`.
  - `index.css`: `.choy-input` height reads `--choy-control-height`.
- **L2 SFCs:**
  - `button/Button.vue`: size map → `h-control` / `h-control-sm` /
    `h-control-lg` / `size-control`.
  - `input/Input.vue`, `combobox/ComboboxInput.vue`, `select/SelectTrigger.vue`,
    `tabs/TabsList.vue`: default height → `h-control`.
  - `select/SelectTrigger.vue`: `focus` → `focus-visible` for the ring.
  - `select/SelectItem.vue`: `pl`/`pr`/`left` → `ps`/`pe`/`start`.
  - `dropdown-menu/DropdownMenuItem.vue`: inset `pl-8` → `ps-8`.
  - `dialog/DialogContent.vue`: close control `right-4` → `end-4`; close focus →
    `focus-visible:ring-*` (centering stays physical `left-1/2` + translate).
  - `toast/Toaster.vue`: viewport `right-0` → `end-0`; ToastClose ring →
    `focus-visible:ring-*`.
  - `switch/Switch.vue`: checked thumb uses mutually exclusive
    `ltr:data-[state=checked]:translate-x-4` /
    `rtl:data-[state=checked]:-translate-x-4`.
  - `index.ts`: note that domain modules must not import this tree directly.
- **Do not regress:** `rg 'h-9|h-10|\bh-8\b' modules/web/web/components/vendor/ui`
  should stay empty for control-height classes (Switch track `w-9` is unrelated).
  Go guard: `TestControlHeightThemeUtilitiesEmit` (theme must still emit
  `h-control` / `size-control`).
### 2026-09-22 — initial hand-adapted import

- **Recorded in:** `VENDOR.json` (`copiedAt`, `source: shadcn-vue (hand-adapted)`).
- L2 `data-slot` kept; colors via `--choy-*` / theme aliases; chart shell for
  Unovis; no Element Plus / echarts / shadcn CLI. Exact per-file class diffs from
  upstream at copy time were not itemized here.

## 2026-10-03 — W2 product wiring (AlertDialog / Sonner / Label / BreadcrumbLink)

- Product hosts now mount AlertDialog (Confirm) and Sonner (ChoyMessage).
- QuickJS FE: inlined local `defineProps` on AlertDialog*, Sonner, Label,
  BreadcrumbLink (same rule as Sidebar* — no `defineProps` from `reka-ui` /
  `vue-sonner` type imports).

## 2026-10-03 — W3 product wiring (Bubble / Message / Item / Attachment / AspectRatio)

- Product hosts mount Bubble/Message (Chatter), Attachment (Binary/Image), Item
  (Notification inbox), AspectRatio (image preview).
- QuickJS FE: inlined local `as`/`asChild` (and AspectRatio `ratio`) on those
  vendor SFCs — no `defineProps` from `reka-ui` type imports.
