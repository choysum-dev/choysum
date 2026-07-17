# Web module terminology (gettext)

Catalogs here are the **packaged terminology** for the `web` application.

- `web.pot` — extract template (`choysum i18n extract web`)
- `zh_CN.po` — language file (`choysum i18n sync web --lang zh_CN`)
- CI gate: `choysum i18n status web --lang zh_CN` (or `scripts/ci/i18n_status.sh`)

Runtime reads terms via Go `web.I18n` → host Gateway → frontend `createTranslate` / vue-i18n.

Extraction is explicit and literal-only: `_t`, backend `_lt`, and `_tr` calls with
string-literal messages are written to POT. Vue labels, selections, menus, routes,
actions, and other static metadata are not extracted and remain English source text.
Frontend code has no `_lt`; backend `_lt` is reserved for static declarations that
must resolve lazily at request/render time.

Language switch (D9 / S6): default `location.reload`. Soft remount is experimental
(`choysum.web.i18n.remountMode=remount`) and only clears menu/global scoped stores.

Optional admin PO download: `GET /web/i18n/po?lang=&application=` (Terminology Editor).

The handwritten packs under `web/web/i18n/source` and `web/web/i18n/translations/` are **legacy coexistence** only (English msgid baseline / historical zh-CN). New shell copy must use `_t` + this PO tree.
