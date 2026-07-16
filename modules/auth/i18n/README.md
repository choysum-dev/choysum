# Auth module terminology (gettext)

- `auth.pot` / `zh_CN.po` — packaged terms for auth service errors and Login UI.
- Runtime: Go `auth.I18n` + Gateway; FE uses `createFeTranslate('auth')`; service uses `createTranslate('auth')`.
