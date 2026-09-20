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
internals). **This skill is authoritative on clean checkouts.**

**Before changing public static methods or request/response envelopes, read this
skill.** Hard-cut execution is separate and may live only in local maintainer
docs (see below).

Optional long-form under gitignored `.dev/` (skip if absent; not in git):

| Path | Read when (local only) |
| --- | --- |
| `.dev/docs/core/service/business_module_service_api_principles.md` | Extra author norms beyond this skill |
| `.dev/docs/core/service/business_module_service_api_review20260920.md` | Context on current gaps vs Odoo |
| `.dev/docs/core/service/business_module_service_api_hardcut_plan20260920.md` | No-compat migration waves (`PR-W1`…`PR-W5`) |
| `.dev/docs/infra/cross_app_service_import_boundary_plan20260901.md` | Dial vs value-import across applications |
| `.dev/docs/core/service/orm/basemodel-author-api-design.md` | CRUD / env axes on BaseModel |

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

- Copy `modules/document/service/contracts.ts` patterns into other domain modules (partner/sale/…).
- Dual auth façades: clients use `auth.User` only; not `auth.Token.*` as a public session API.
- ≥3 business positional parameters on a public method.
- Entity DTO packages or CreateXxxDto + Mapper layers.

## Review output

When reviewing a change, report:

- Which hard rules apply and any violations (with method / type names).
- Whether new public verbs should be envelopes, CRUD-only, or worker-private.
- If a migration is required, name the hardcut wave below (or describe the step
  in prose when local hardcut docs are absent):
  - `PR-W1` — workers off public RPC; identity from session; `auth.Token` not client-facing
  - `PR-W2` — `auth.User` session verbs → single Req envelopes (+ Register result shape)
  - `PR-W3` — task/meta/document/message envelopes, Schedule single write path, naming
  - `PR-W4` — docs + core boundary helpers (`Normalized*` not exported)
  - `PR-W5` — domain additions (UoM.Convert, Partner FindOrCreate/Lookup, LanguageId)

## Scope guardrails

- Primary surface: `modules/*/service/models/**/*.ts` and exported service entrypoints.
- Skip generated protobuf/clients unless the user asks to update call sites after a hardcut.
- `modules/core/service/**` platform changes follow BaseModel / protobuf docs, not this skill’s domain author checklist alone.
