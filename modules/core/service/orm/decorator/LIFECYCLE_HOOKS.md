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
4. Treat handlers as **module install/upgrade/uninstall callbacks** or **process
   ready callbacks**, not Record create/write lifecycle hooks.
5. `@Migration` requires both `version` and a valid `phase` (`'pre' | 'post' | 'end'`);
   missing or invalid options fail with `LIFECYCLE_MIGRATION_INVALID_OPTIONS`.
6. Decorate **methods**, not property/field initializers. Unresolvable method functions
   fail with `LIFECYCLE_HOOK_INVALID_METHOD` / `LIFECYCLE_MIGRATION_INVALID_METHOD`.

## Phases

### Evolution (install / upgrade / uninstall)

`pre_init` / `post_init` / `pre_upgrade` / `post_upgrade` / `pre_uninstall` /
`post_uninstall` run during module evolution. **Do not call cross-app RPCs** here
(for example `task.Schedule` via `createServiceByModel`): during bootstrap install
the application service registry is not available.

### Ready (process serving)

`@HookReady()` / phase `ready` runs after **this process** has registered its
application services, activated the JS runtime, and started transport ingress
(so cross-app gRPC Dial works). Triggered on cold start, bootstrap mode-switch
into application, and equivalent reload. Use this for cross-app seeds such as
ensuring a `task.Schedule` row. The phase is **required** by default: failures
block startup readiness. Recovery: fix the hook (or its seed data), then restart
the process; there is no skip flag — a permanently failing ready hook keeps the
process from becoming ready until the underlying error is resolved.

Application init evaluates each backend module under a per-module `CHOYSUM_*`
seed and installs a resident `__choysum_hook__` dispatcher. Ready therefore
**invokes** already-registered handlers on the shared runtime (no Bundle /
`SetJsScripts` / Reload). Cross-app calls prefer the in-process JS service pool;
when Dial is required, host `InternalKey` / mTLS may call a narrow allowlist
(e.g. `task.Schedule/Search|Create|UpdateById`) — not interactive clients.

Without a request session, ready / evolution JS context uses the bootstrap
admin principal (`auth.user_admin` → real `User.Id` + company scope), not the
Username literal `"admin"`. Record rules still apply (sys.admin global grant);
this is not sudo / `recordRuleMode=skip`.

## Canonical sample (cross-app schedule ensure)

`modules/document/service/hook/ready.ts` (`DocumentAttachmentHooks`).

```ts
const ScheduleService = createServiceByModel<typeof Schedule>('task.Schedule');

export class DocumentAttachmentHooks {
  @HookReady()
  static async ensureAttachmentGcSchedule(): Promise<void> {
    // use ScheduleService / module helpers — never this
  }
}
```

Document attachment GC execution: Schedule `FullMethod` is a **TaskWorker
routing key** (`document.AttachmentContent/RunGarbageCollection`). The model
method is intentionally **not** `public` (not interactive gRPC); TaskWorker
invokes it in-process via an allowlist after `TaskWorker/ExecuteJob`.

## Forbidden

```ts
export class Bad {
  @HookPostInit()
  async ensure(): Promise<void> {} // instance method

  @HookReady()
  static async ensure2(): Promise<void> {
    this.doSomething(); // static but still relies on this
  }
}
```

## Contrast: Record lifecycle (`@Onchange` / `@Constraint` / `@Compute`)

Hook/Migration handlers are **module-level** and must not use `this`.

Record create/write lifecycle handlers are the opposite: they **require** a correct
draft or bridge `this`, and may call **class-level** Model APIs such as
`OtherModel.Search(...)`. Do not apply “never use `this`” or “never call Model
RPC” from this Hook doc to those handlers.

See `modules/core/service/runtime/RECORD_LIFECYCLE_THIS.md`.
