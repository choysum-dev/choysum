# Core module terminology (gettext)

- `core.pot` / `zh_CN.po` — packaged terms for shared platform validation,
  authz denials, CRUD not-found messages, and web error fallbacks owned by core.
- CI gate: `choysum i18n status core --lang zh_CN`.
- **Scheme A (runtime):** there is no `core.TranslationTerm` / `core_translation_term`.
  On install/upgrade of each real Application module, Go imports
  `modules/core/i18n/*.po` into that app's `{app}_translation_term` with
  `Module=core`. Upgrading `core` itself fans the same PO out to every host app.
  Gateway dials each `{app}.TranslationTerm.GetTranslations` for module `core`
  alongside the app's own modules.
- Service code imports `_t` / `_lt` from `modules/core/service/i18n_binder.ts`
  (`createTranslate('core')`). Frontend fallbacks in `core/web` use
  `createTranslate('core', …)` from `@/core/service/i18n` (browser resolves via
  `$choysum.i18n.t` installed in `web/app.ts`).
- Core owns the shared i18n **platform** under `service/i18n/`; keep that package
  free of module-owned `_t('…')` / `_lt('…')` literals so catalogs stay cross-cutting only.
- Core has no domain menus/routes/views — Ir* chrome lives in `modules/meta`.
- Only literal `_t` / `_lt` calls are extracted. Selection labels stay plain English
  until a request-scoped options API exists (do not use `_lt` for selection labels).
- Keep `"application": "core"` in `package.json` as the D13 sentinel (skip DDL /
  TranslationTerm host registration). Do not remove or empty it without relaxing
  `ValidatePackageJSON` and updating all skip checks.
