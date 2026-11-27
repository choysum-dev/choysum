<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

# Choy view sheet

Paper surface contract for page views (Form / List / Kanban / Chart) under the
app shell. Shell canvas is a muted pane; the floating white paper is
`view-body` (`choy.view.sheet`).

## Layers

| Layer | Owns | Does not own |
| --- | --- | --- |
| shell canvas | muted fill, scroll | content gutter |
| `view-header` | Action Bar / statusbar; same gutter `mx` as the paper | paper fill / radius |
| `view-body` | paper shell: `bg-card`, radius, shadow, hairline border, **margin** gutter | content padding |
| `view-body-host` | in-paper **padding** density (`sheetPad`) | outer margin / second card |
| domain content | fields, table, board | page-level gutter |

Vertical air between Action Bar and paper is container `gap`. Page Form / List /
Kanban / Chart use `gap-2` (same rhythm as sheet gutter / `section-gap`), not
margin on `view-body`. ViewBase’s default `gap-5` remains for non-sheet hosts;
sheet pages override via their root `class`.

## Tokens

Defined in `modules/web/web/styles/tokens.css` (compact overrides gutter/pad):

- `--choy-view-sheet-gutter` — paper vs canvas (`section-gap`, same as nav-pane end margin)
- `--choy-view-sheet-pad` — in-paper density for `sheetPad=form` (aliases content-padding)
- `--choy-view-sheet-radius` — `--choy-radius-md` (view-body uses `rounded-md`, same as nav-pane)

## ViewBase API

```ts
sheetSurface?: boolean; // default true; false for embedded / dialog
sheetPad?: 'form' | 'board' | 'flush' | 'none'; // default 'flush'
```

### Unification matrix

Two in-paper densities: **document / card-canvas** (`form`) vs **dense collection**
(`board`, List/Chart). Shell chrome (gutter, container `gap-2`, paper
radius/border) is shared. Kanban uses `form` so tiles clear the paper edge.

| View | sheetSurface | sheetPad | Container gap | header-extra |
| --- | --- | --- | --- | --- |
| Form page / settings | on | `form` (four-side `--choy-view-sheet-pad`) | `gap-2` | statusbar; no own py |
| Form embedded | off | `none` | ViewBase default | — |
| List | on | `board` (four-side `--choy-layout-section-gap`) | `gap-2` | — |
| Kanban | on | `form` (card grids need paper-edge air like Form) | `gap-2` | — |
| Chart | on | `board` | `gap-2` | controls; no own py (header gap only) |

`flush` remains on the ViewBase API for rare full-bleed hosts; page List /
Kanban / Chart do not use it.

## Rules

1. Outer gutters are **margin** on `view-body` (and matching `view-header`), never
   host padding — otherwise the paper fills the canvas and rounded corners lose
   the pane reveal.
2. In-paper density is **padding** on `view-body-host` only (`form` / `board`).
   Form and Kanban use `form`; List and Chart use denser `board`.
3. Do not reintroduce Form `border-t` as a fake paper edge; the paper uses
   `rounded-md` + `shadow-xs` + a hairline `border-border` so edges stay readable
   when pane ≈ card white.
4. Embedded views must set `sheetSurface=false` and `sheetPad=none` so dialogs
   do not double-frame.
5. Chart / Form `header-extra` must not add vertical padding; Action Bar ↔ extra
   ↔ paper rhythm is view-header `gap-2` only.
6. Kanban nested radius: sheet stays `rounded-md` (`--choy-view-sheet-radius`).
   In-paper lanes (`bg-muted` wash) and cards (plus dashed empty placeholders)
   use `rounded-sm` so they stay smaller than the paper. Do not nest the same
   radius as the sheet (md on lane/card). Relation-field kanban cards follow the
   same card/`empty` step-down.
