# Auth module terminology (gettext)

- `auth.pot` / `zh_CN.po` — packaged terms for auth service errors and Login UI.
- CI gate: `choysum i18n status auth --lang zh_CN`.
- Runtime: Go `auth.I18n` + Gateway; frontend and service runtimes each expose `createTranslate('auth')`.
- Frontend Gateway catalogs are projected into vue-i18n's flat `__terms`
  namespace. Serializable `_td` metadata carries the deterministic full-identity
  key used by Composer `te`/`t`; English `src` remains the pre-load/missing-term
  fallback. Service `_t`/`_lt` remain backed by the Go bridge.
