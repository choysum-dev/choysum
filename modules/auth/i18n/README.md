# Auth module terminology (gettext)

- `auth.pot` / `zh_CN.po` — packaged terms for auth menus, routes, CRUD views,
  Login/Register/Logout chrome, store actions, and service error messages.
- CI gate: `choysum i18n status auth --lang zh_CN`.
- Runtime: Go `auth.I18n` + Gateway; frontend and service runtimes each expose `createTranslate('auth')`.
- `createTranslate('auth')` exposes only `_t`. Text output is the default and the
  service implementation remains backed by the Go bridge. Frontend text output
  uses vue-i18n and `computed(() => _t(...))` for reactive values.
- `createTranslate('auth', { output: 'reference', scope: '...' })` returns
  deterministic serializable term references without lookup. Use this for menus,
  routes, and `defineModelActions` `entityTitle` (so Create/Edit/Delete/Copy
  terms are synthesized at extract time). Gateway catalogs are projected into
  vue-i18n's flat `__terms` namespace; English `src` remains the
  pre-load/missing-term fallback.
- `_t(..., { output: ... })` overrides the factory output. Reference calls
  reject interpolation; text calls support it. Output is not part of term
  identity and cannot change deterministic keys or catalog entries.
- Only literal `_t` calls are extracted. Explicit closures provide any
  process-local lazy behavior.
- Selection field labels (Allow/Deny/…) stay plain English until a request-scoped
  options API exists.
