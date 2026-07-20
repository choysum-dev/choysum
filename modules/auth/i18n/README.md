# Auth module terminology (gettext)

- `auth.pot` / `zh_CN.po` — packaged terms for auth menus, routes, CRUD views,
  Login/Register/Logout chrome, store actions, and service error messages.
- CI gate: `choysum i18n status auth --lang zh_CN`.
- Runtime: Go `auth.I18n` + Gateway; frontend and service runtimes each expose
  `createTranslate('auth')` → `{ _t, _lt }`.
- `_t` — immediate text (service Go bridge / FE vue-i18n). Use for errors and
  reactive UI copy (`computed(() => _t(...))`).
- `_lt` — serializable `TermReference` without lookup. Use for menus, routes,
  and `defineModelActions` `entityTitle` (so Create/Edit/Delete/Copy terms are
  synthesized at extract time). Gateway catalogs are projected into vue-i18n's
  flat `__terms` namespace; English `src` remains the pre-load/missing-term
  fallback.
- Only literal `_t` / `_lt` calls are extracted. Explicit closures provide any
  process-local lazy behavior.
- Selection field labels (Allow/Deny/…) stay plain English until a request-scoped
  options API exists (do not use `_lt` for selection labels — D5).
