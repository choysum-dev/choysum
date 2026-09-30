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
