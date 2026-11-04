# Modules

TypeScript modules for the Choysum ERP platform (service + web entry points).

## Data seed ownership

Preset rows live in each module’s `data/` (install) and optional `demo/` (`--with-demo`).

1. **Master / business rows** → module that owns the model (e.g. `base.Company` → `base/data`).
2. **Domain-targeted authz** → the **domain** module that owns the model (or domain-owned logical name). Seed `auth.Role*` with `application: "auth"`; xml_id stays under the applying module (example: `web` bootstrap → `web.UserFilter` record rules).
3. **Platform roles, global break-glass, and platform LogicalModel defaults** → `auth/data` only (`base.user`, `sys.admin`, global grants, auth User/Token/Session packs, and `FieldDefault` / `AppSetting` logical RMA/RFR). Terminology UI defaults to **sys.admin** (no global LogicalModel `TranslationTerm` gift pack); platform terms use `core.TranslationTerm`.
4. Domain modules may seed into auth models only if they install **after** auth (`depends: ["auth", …]` and auth does not depend on them). `base` / `meta` install before auth — their app-level gift packs remain in auth until a late-apply path exists.
5. **Do not** add new **domain-model** RR/RFR/RMA into `auth/data`; follow the web UserFilter pattern. Platform logical defaults (item 3) stay in auth.

### Web SPA shell

Do **not** list `web` in domain `depends`. The web shell depends on `core` / `auth` / `meta`; binary/image attachments need optional `document` (CLI `--web-shell-profile=standard|full`, default `standard`). The planner pulls the shell when any module in the install/upgrade plan declares `entryPoints.web`:

- **Install:** merges `web` (and its depends) into `ModuleOrder` if web is not already installed; then Ensures profile extras (`document`, and for `full` also `message`/`audit`).
- **Upgrade:** if web is missing, installs it via `EnsureOrder` (does not upgrade an already-installed web); always rebuilds `dist/web` when needed.
- **`--no-web`:** skip that auto-include (headless / API-only installs).
- **`--web-shell-profile=minimal`:** shell only — no document/message/audit (ProvisionMinimal).

Loader notes: `module` must be the applying module; `application` may be cross-app; `model` is a short name; use `ref` / `modelRef` / `refBy` for links.
