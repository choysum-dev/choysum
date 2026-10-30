# Core module terminology (gettext)

- `core.pot` / `zh_CN.po` — packaged terms for shared platform validation,
  authz denials, CRUD not-found messages, and web error fallbacks owned by core.
- CI gate: `choysum i18n status core --lang zh_CN`.
- **Scheme B (runtime):** platform terms live in `core_translation_term` /
  `core.TranslationTerm`. On install/upgrade of the `core` module, Go imports
  `modules/core/i18n/*.po` into that table only (no fan-out to host apps).
  Gateway dials `core.TranslationTerm/GetTranslations` alongside each business
  app; host app tables hold only that app's module terms.
- **Upgrade from Scheme A:** existing `{app}_translation_term` rows with
  `Module=core` are stale under Scheme B (platform catalog is `core` only).
  Upgrade cleanup should delete or ignore those rows so lookups do not resolve
  duplicate/outdated platform terms from host app tables.
- Service code imports `_t` / `_lt` from `modules/core/service/i18n_binder.ts`
  (`createTranslate('core')`). Frontend fallbacks in `core/web` use
  `createTranslate('core', …)` from `@/core/service/i18n` (browser resolves via
  `$choysum.i18n.t` installed in `web/app.ts`).
- Core owns the shared i18n **platform** under `service/i18n/`; keep that package
  free of module-owned `_t('…')` / `_lt('…')` literals so catalogs stay cross-cutting only.
- Terminology UI: Settings → {app} → Translations (folders, pages, and routes in `web`;
  each installed app registers its Settings folder; `core` / `web` hosted by `web`).
  Route `/web/terminology/core` (sys.admin by default).
- Only literal `_t` / `_lt` calls are extracted. Selection labels stay plain English
  until a request-scoped options API exists (do not use `_lt` for selection labels).
- `"application": "core"` in `package.json` is the framework application name and
  TranslationTerm host; injectappmodel still skips FieldDefault / AppSetting /
  PropertyDefinition for core.
