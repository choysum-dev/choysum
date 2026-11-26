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

Vertical air between Action Bar and paper is container `gap` (page Form uses
`gap-2`, matching sheet gutter / `section-gap`), not margin on `view-body`.

## Tokens

Defined in `modules/web/web/styles/tokens.css` (compact overrides gutter/pad):

- `--choy-view-sheet-gutter` — paper vs canvas (`section-gap`, same as nav-pane end margin)
- `--choy-view-sheet-pad` — in-paper density (aliases content-padding)
- `--choy-view-sheet-radius` — `--choy-radius-md` (view-body uses `rounded-md`, same as nav-pane)

## ViewBase API

```ts
sheetSurface?: boolean; // default true; false for embedded / dialog
sheetPad?: 'form' | 'flush' | 'none'; // default 'flush'
```

| Preset | Surface | Host pad | Used by |
| --- | --- | --- | --- |
| page Form / settings | on | `form` (four sides) | `ChoyFormView` when not embedded |
| embedded Form | off | `none` | dialogs / nested forms |
| List / Kanban / Chart | on | `flush` | tables and boards edge-to-edge inside the paper |

## Rules

1. Outer gutters are **margin** on `view-body` (and matching `view-header`), never
   host padding — otherwise the paper fills the canvas and rounded corners lose
   the pane reveal.
2. In-paper density is **padding** on `view-body-host` only (`sheetPad=form`).
3. Do not reintroduce Form `border-t` as a fake paper edge; the paper uses
   `rounded-md` + `shadow-xs` + a hairline `border-border` so edges stay readable
   when pane ≈ card white.
4. Embedded views must set `sheetSurface=false` and `sheetPad=none` so dialogs
   do not double-frame.
