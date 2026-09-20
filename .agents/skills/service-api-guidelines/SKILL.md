---
name: service-api-guidelines
description: >-
  House rules for Choysum business-module Service APIs (model-as-service, thin
  command envelopes, identity from session, RPC discovery, codec honesty). Use
  when writing, reviewing, or refactoring modules/*/service/** (especially
  models and public static PascalCase async methods); when adding XxxReq/Params
  types; when exposing or hiding gRPC verbs; or when asked about Service API
  design vs Odoo / DTO / auth.User vs Token.
---

# Business module Service API guidelines

House rules for `modules/*/service/**` (business modules; not core platform
internals). Canonical long-form norms live under `.dev/docs/core/service/`.

**Before changing public static methods or request/response envelopes, read this
skill and the linked principles.** Hard-cut execution is separate.

| Document | Read when |
| --- | --- |
| [Principles](../../../.dev/docs/core/service/business_module_service_api_principles.md) | Authoring or reviewing any Service API change |
| [Review archive 2026-09-20](../../../.dev/docs/core/service/business_module_service_api_review20260920.md) | Context on current gaps vs Odoo |
| [Hardcut plan 2026-09-20](../../../.dev/docs/core/service/business_module_service_api_hardcut_plan20260920.md) | Implementing the no-compat migration waves |
| [Cross-app import boundary](../../../.dev/docs/infra/cross_app_service_import_boundary_plan20260901.md) | Dial vs value-import across applications |
| [BaseModel author API](../../../.dev/docs/core/service/orm/basemodel-author-api-design.md) | CRUD / env axes on BaseModel |

Related skills: [`module-initdata`](../module-initdata/SKILL.md) for authz seed placement; do not confuse init-data ownership with Service RPC shape.

## Motto

1. **CRUD uses the model** — `Insertable` / `FieldSelection`; no Entity DTO/Mapper.
2. **Actions use a thin envelope** — one `XxxReq|Params` → `XxxResp|Result`.
3. **Identity from session** — never trust `principal` / client actor ids on the wire.
4. **Internals stay off RPC** — workers/helpers must not be `public static` PascalCase async.
5. **Outbound via wrapper, inbound via assert** — platform representation-normalize only.

## Hard rules

1. **Model is the API** — public contract is `{application}.{Model}` methods.
2. **RPC discovery** — only `public` + `static` + PascalCase + `async` becomes gRPC.
3. **Actions** — ≤1 business object argument (optional `fields?: FieldSelection` OK); no public `unknown` / `Record<string, unknown>` as primary contract.
4. **Identity** — resolve uid/company from session; sudo/worker overrides only with ACL.
5. **Async ops** — `Request*` for interactive users; `Execute*` / `FanOut*` / `RunGarbageCollection` for workers only (or non-discoverable shape).
6. **Codec honesty** — authors `assert*` / `parseDecimalInput` / `toDate` at the boundary; do not assume inbound `Date`/`Decimal`/`BaseModel`.
7. **Same-app model merge** — re-`@Model('Name')` with the same `choysum.application`; cross-app use `dial` / `createServiceByModel`, never value-import.

## Naming

- Compute-like: `Params` / `Result`. Command-like: `Req` / `Resp`.
- Envelope fields: PascalCase (align with model fields).
- `Normalized*` stays internal — do not export to clients.

## Do not

- Copy `document/contracts.ts` patterns into domain modules (partner/sale/…).
- Dual auth façades: clients use `auth.User` only; not `auth.Token.*` as a public session API.
- ≥3 business positional parameters on a public method.
- Entity DTO packages or CreateXxxDto + Mapper layers.

## Review output

When reviewing a change, report:

- Which hard rules apply and any violations (with method / type names).
- Whether new public verbs should be envelopes, CRUD-only, or worker-private.
- Pointers to the principles section or hardcut wave if a migration is required.

## Scope guardrails

- Primary surface: `modules/*/service/models/**/*.ts` and exported service entrypoints.
- Skip generated protobuf/clients unless the user asks to update call sites after a hardcut.
- `modules/core/service/**` platform changes follow BaseModel / protobuf docs, not this skill’s domain author checklist alone.
