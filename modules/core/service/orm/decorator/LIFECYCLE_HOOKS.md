<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

# Lifecycle Hook & Migration Author Contract

Short contract for `@Hook*` / `@Migration`. Full design notes live in
`.dev/docs/core/service/lifecycle_hook_this_boundary_plan20260715.md` (local planning tree).

## Rules

1. Decorate **`static` methods only**. Instance methods fail at registration with
   `LIFECYCLE_HOOK_INSTANCE_METHOD_FORBIDDEN` or
   `LIFECYCLE_MIGRATION_INSTANCE_METHOD_FORBIDDEN`.
2. **Do not use `this`** inside handlers. The runner calls `await fn()` with no
   bound receiver.
3. Resolve dependencies via module-level `import` and `createServiceByModel(...)`
   (or other module-scope helpers), not via instance fields or IOC.
4. Treat handlers as **module install/upgrade/uninstall callbacks**, not Record
   create/write lifecycle hooks.
5. `@Migration` requires both `version` and a valid `phase` (`'pre' | 'post' | 'end'`);
   missing or invalid options fail with `LIFECYCLE_MIGRATION_INVALID_OPTIONS`.
6. Decorate **methods**, not property/field initializers. Unresolvable method functions
   fail with `LIFECYCLE_HOOK_INVALID_METHOD` / `LIFECYCLE_MIGRATION_INVALID_METHOD`.

## Canonical sample

`modules/document/service/hook/post_init.ts` (`DocumentAttachmentHooks`).

```ts
const ScheduleService = createServiceByModel<typeof Schedule>('task.Schedule');

export class DocumentAttachmentHooks {
  @HookPostInit()
  static async ensureAttachmentGcSchedule(): Promise<void> {
    // use ScheduleService / module helpers — never this
  }
}
```

## Forbidden

```ts
export class Bad {
  @HookPostInit()
  async ensure(): Promise<void> {} // instance method

  @HookPostInit()
  static async ensure2(): Promise<void> {
    this.doSomething(); // static but still relies on this
  }
}
```
