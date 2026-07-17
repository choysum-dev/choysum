# Web module terminology (gettext)

Catalogs here are the **packaged terminology** for the `web` application.

- `web.pot` — extract template (`choysum i18n extract web`)
- `zh_CN.po` — language file (`choysum i18n sync web --lang zh_CN`)
- CI gate: `choysum i18n status web --lang zh_CN` (or `scripts/ci/i18n_status.sh`)

Runtime reads terms via Go `web.I18n` → host Gateway → frontend vue-i18n.

`_td` returns a serializable descriptor containing `module`, `scope`, `src`,
`kind`, and a deterministic `key`. The key is `__terms.<hex identity>` where the
hex segment encodes the UTF-8 byte-length-prefixed full identity
`module/scope/src/kind`; it contains no dots and is identical in frontend,
backend parser, and generated metadata.

Before Gateway messages are merged, the legacy nested
`module → scope → src → value` catalog is preserved and also projected into the
flat vue-i18n `__terms` namespace. Vue templates display descriptors directly
with `$t(descriptor.key, descriptor.src || fallback)` (or a plain-string branch
when the descriptor is optional). TypeScript, h-render, and router consumers use
the single `translateTerm(composer, descriptor, fallback)` adapter. The adapter
relies on vue-i18n's default-message and locale-fallback behavior, so it does not
preflight with `te`. Missing and not-yet-loaded catalogs therefore render the
English source or legacy fallback. Backend `_t`/`_lt` continue to use the Go
bridge; vue-i18n is frontend-only.

Extraction is explicit and literal-only: `_t`, backend `_lt`, `_tr`, and `_td`
calls with string-literal messages are written to POT. Static descriptor
consumers retain English source text as their fallback.
Frontend code has no `_lt`; backend `_lt` is reserved for static declarations that
must resolve lazily at request/render time.

Language switch (D9 / S6): default `location.reload`. Soft remount is experimental
(`choysum.web.i18n.remountMode=remount`) and only clears menu/global scoped stores.

Optional admin PO download: `GET /web/i18n/po?lang=&application=` (Terminology Editor).

The handwritten packs under `web/web/i18n/source` and `web/web/i18n/translations/` are **legacy coexistence** only (English msgid baseline / historical zh-CN). New shell copy must use `_t` + this PO tree.
