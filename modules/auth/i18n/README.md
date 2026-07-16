# Auth module terminology (gettext)

- `auth.pot` / `zh_CN.po` — packaged terms for auth service errors and Login UI.
- CI gate: `choysum i18n status auth --lang zh_CN`.
- Runtime: Go `auth.I18n` + Gateway; FE uses `createFeTranslate('auth')`; service uses `createTranslate('auth')`.
