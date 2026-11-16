<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

# base module init data

| File | Contents |
| --- | --- |
| `bootstrap.json` | Default-active `language_zh_cn` / `language_en_us`, `currency_cny`, `company_main`. |
| `languages.json` | Inactive Odoo-style Language catalog (~90 rows). **Generated** — do not hand-edit. |
| `currencies.json` | Currency catalog (mostly inactive). |
| `countries.json` | Country catalog. |

## Language catalog

- Source CSV: `scripts/base/data/res.lang.csv` (vendored from Odoo `res.lang`).
- Generator: `python3 scripts/base/generate_language_seeds.py --write`
- Drift check: `python3 scripts/base/generate_language_seeds.py --check`
- Codes containing `@` (e.g. `sr@latin`) are skipped.
- `en_US` / `zh_CN` stay in `bootstrap.json` as the default-active authority; `languages.json` omits them and ships every other row with `IsActive: false`.

**Enabling a language (`IsActive`) does not install PO / terminology packs.** Activating a row makes it selectable in the UI; translations still come from Gateway / msgid fallback until packs exist.
