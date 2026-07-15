<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

# Record Lifecycle `this` vs Service Wrapper Author Contract

Short contract for `@Onchange` / `@Constraint` / `@Compute` and
`$sql` / `$search` / `$inverse` handlers.

Full design notes:
`.dev/docs/core/service/record_lifecycle_proxy_wrapper_boundary_plan20260715.md`

This is the **opposite** of the Hook/Migration contract
(`modules/core/service/orm/decorator/LIFECYCLE_HOOKS.md`):

| Concern | Hook / Migration | Record lifecycle (this doc) |
| --- | --- | --- |
| Host | Module-level `static` | Instance / draft / bridge |
| `this` | **Must not** rely on `this` | **Must** use the correct `this` |
| Model API | Prefer `createServiceByModel` | Class-level `Model.Static(...)` is allowed |

## Rules

1. Inside `@Onchange` / `@Constraint` / `@Compute`, `this` is the session draft
   (or compute model instance). Use it for **field** reads/writes only.
2. Inside `@Search` / `@SqlCompute` / `@Inverse`, `this` is the bridge
   **execution instance**. Use only `this.$search` / `this.$sql` / `this.$inverse`.
3. Cross-model reads use **class-level** Model API (same idea as Odoo
   `self.env['account.move.line'].search_count(...)` in
   `odoo/addons/account/models/account_account.py` constraints/computes):

   ```ts
   await OtherModel.Search(...);
   await OtherModel.Count(...);
   await Currency.Convert(...);
   ```

4. Never bind a draft into a conventional service method:

   ```ts
   // FORBIDDEN
   await OtherModel.Search.call(this, ...);
   await OtherModel.Search.apply(this, [...]);
   await OtherModel.Search.bind(this)(...);
   await this.Search(...);
   await this.constructor.Create(...);
   ```

5. Onchange preview forbids current-record persistence via draft `this`
   (`update` / `delete` / `reload`, conventional CRUD/query names, and common
   Odoo/ORM aliases such as `write` / `unlink` / `save`). Constraint drafts use
   the same persistence guard. Class-level reads remain allowed.
6. One call stack, one `this` identity. Do not mix Onchange/Constraint drafts
   with `$search` / `$sql` / `$inverse`, and do not wrap drafts with another
   hydrate/bridge proxy.

## Hard failures (runtime)

| Misuse | Error prefix |
| --- | --- |
| `Model.Static.call(draft, ...)` | `SERVICE_WRAPPER_INVALID_THIS` |
| `this.update` / `this.Create` on onchange draft | `PREVIEW_METHOD_FORBIDDEN` |
| Same persistence methods on constraint draft | `CONSTRAINT_DRAFT_METHOD_FORBIDDEN` |
| `$search` / `$sql` / `$inverse` on a draft | `BRIDGE_CONTEXT_*` |

## Allowed

```ts
@Constraint<Account>(['CurrencyId'])
async checkJournalConsistency() {
  // this = constraint draft (field semantics)
  const n = await AccountMoveLine.Count({
    condition: [['AccountId', '=', this.Id as any]],
  });
  // ...
}

@Onchange<Account>('AccountType')
onchangeAccountType() {
  if (this.AccountType === 'off_balance') {
    this.TaxIds = [] as any;
  }
  // Need other model data: OtherModel.Browse / Search (class-level).
}
```

## Forbidden

```ts
@Onchange<Account>('Name')
async bad() {
  await Account.Search.call(this as any, []); // SERVICE_WRAPPER_INVALID_THIS
  await this.update({} as any); // PREVIEW_METHOD_FORBIDDEN
  void this.$search; // BRIDGE_CONTEXT_*
}
```

## Implementation anchors

- Proxy brands: `modules/core/service/runtime/proxy/brand.ts`
- Wrapper `thisArg` guard: `modules/core/service/orm/decorator/service.ts`
- Onchange invoke: `modules/core/service/runtime/onchange/engine.ts`
- Constraint draft: `modules/core/service/runtime/validation/engine.ts`
- Bridge identity: `modules/core/service/runtime/compute/bridge.ts`
