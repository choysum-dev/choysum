<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

# View workflow actions (`useFormWorkflowAction` / `useListWorkflowAction`)

Domain buttons that call a **side-path RPC** (not Form Save / List platform Delete) must
resync the view surface afterward. Action kit chrome
(`ChoyActionButton` / `ChoyActionMenuItem` / `ChoyStatusbarButton`) stays an L3 shell
(ACL / confirm / offline / busy / `run|execute`) and **does not** call store methods by
name or run `reloadDisplay` / list apply.

Use:

- `useFormWorkflowAction` — Form `#statusbar-buttons` after Activate / Mark Open / …
- `useListWorkflowAction` — List `#contextual-actions` after bulk domain writes

Platform **Refresh** (Form/List overflow) is **not** this flow: Refresh *is* the surface
reload (`handleRefresh` → `beginDisplay` / `apply` + toast). Domain CTA = `call` then
optional silent sync; Refresh = user-requested reload.

KeepAlive breadcrumb return uses `useDetailPathCollectionSoftRefresh` (orthogonal to
in-page workflow RPC).

## Host migration matrix

| Bucket | Hosts | Action |
| --- | --- | --- |
| **MIGRATE_NOW** | Sandbox Form statusbar; Sandbox List `#contextual-actions`; Language Form statusbar (thin: `reload:'none'` + existing `reloadRecord`) | Use composables |
| **CATALOG_LATER** | ModuleCatalog Form statusbar (Install/Upgrade/Uninstall), List/Kanban Sync & Load Demo, row/card Primary/Uninstall, OpDialog Confirm | Dialog + job + `refreshCatalog` / narrow Browse — use `reload:'none'` + `onSuccess` later, or a catalog-specific helper; **not** this slice |
| **OPTIONAL** | Import/Export wizards, field translation dialogs, Chatter, Partner nested O2M Edit | Local state; at most borrow busy |
| **DO_NOT** | `ChoyFormView` / `ChoyListView` / Kanban/Chart platform CRUD; Auth; Notification; Attachment chrome; Overflow Refresh | Platform already owns refresh |

## API

```ts
useFormWorkflowAction({
  formRef, // ChoyFormView expose
  reload: 'display' | 'none', // default 'display'
  onSuccess?,
  errorMessage?,
})
// → { busy, run(call) }

useListWorkflowAction({
  listRef, // ChoyListView expose
  reload: 'apply' | 'none', // default 'apply' → softRefresh
  clearSelection?: boolean, // default false; Sandbox bulk uses true
  onSuccess?,
  errorMessage?,
})
```

- Form `reload:'display'`: display → `reloadDisplay()`; edit/create → no-op (keep draft).
- List `reload:'apply'`: silent `softRefresh()` (re-apply; does not clear selection by itself).

## Sandbox List bulk demo

`SandboxRecordListView` fills `#contextual-actions` with Mark Open / Mark Done / Reset to
Draft (`ChoyActionButton`). Gated by `edit` ACL and selected-row `State`. Writes
`UpdateById({ State, Progress })` per matching id, then
`useListWorkflowAction({ reload: 'apply', clearSelection: true })`.
