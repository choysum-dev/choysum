<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

# base module init data

| File | Contents |
| --- | --- |
| `bootstrap.json` | Default-active `language_zh_cn` / `language_en_us`, `currency_cny`, `company_main`. |
| `languages.json` | Inactive Language catalog (~90 rows). Hand-maintained; initially seeded from Odoo `res.lang` naming. |
| `currencies.json` | Currency catalog (mostly inactive). |
| `countries.json` | Country catalog. |

## Language catalog

- Edit `languages.json` directly when adding or adjusting inactive languages.
- `en_US` / `zh_CN` stay in `bootstrap.json` as the default-active authority; `languages.json` omits them and ships every other row with `IsActive: false`.
- Display names follow the Odoo-style `English / native` string in `Name.en_US` unless a bilingual map is intentional.

**Enabling a language (`IsActive`) does not install PO / terminology packs.** Activating a row makes it selectable in the UI; translations still come from Gateway / msgid fallback until packs exist.
