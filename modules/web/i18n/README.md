# Web module terminology (gettext)

Catalogs here are the **packaged terminology** for the `web` application.

- `web.pot` — extract template (`choysum i18n extract web`)
- `zh_CN.po` — language file (`choysum i18n sync web --lang zh_CN`)

Runtime reads terms via Go `web.I18n` → host Gateway → FE `createFeTranslate` / vue-i18n.

The handwritten packs under `web/web/i18n/source` and `web/web/i18n/translations/` are **legacy coexistence** only (English msgid baseline / historical zh-CN). New shell copy must use `_t` + this PO tree.
