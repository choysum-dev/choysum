# Core module terminology (gettext)

- `core.pot` / `zh_CN.po` — packaged terms for shared platform validation,
  authz denials, CRUD not-found messages, and web error fallbacks owned by core.
- CI gate: `choysum i18n status core --lang zh_CN`.
- Runtime: Go `core.I18n` + Gateway; service code imports `_t` from
  `modules/core/service/i18n_binder.ts` (`createTranslate('core')`).
- Frontend fallbacks (if any) use `createTranslate('core', …)` from
  `@/web/web/i18n`.
- Core owns the shared i18n **platform** under `service/i18n/`; keep that package
  free of module-owned `_t('…')` literals so catalogs stay cross-cutting only.
- Core has no domain menus/routes/views — Ir* chrome lives in `modules/meta`.
- Only literal `_t` calls are extracted. Selection labels stay plain English
  until a request-scoped options API exists.
