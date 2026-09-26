<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

# Vendored kit primitives

`ui/` holds hand-adapted shadcn-vue L2 SFCs (Reka + `--choy-*` tokens). Not a public
product API — domain modules must use exported `Choy*` only.

**SFC block order:** keep shadcn's `script` → `template` here so upstream diffs stay
small. Every other product SFC — `Choy*`, domain views/fields, `internal/*`, and
`modules/*/web/pages/*.vue` — uses `template` → `script` → `style` (IMD/xpath
reviewers open the markup first).

**Cutover:** this tree moves with the kit into `modules/web/web/components/vendor/`
(same relative layout). Keep the `vendor/ui` segment stable so ignore rules, import
bans, and upstream diffs do not need a second rename.
