<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

# Vendored kit primitives

`ui/` holds hand-adapted shadcn-vue L2 SFCs (Reka + `--choy-*` tokens). Not a public
product API — domain modules must use exported `Choy*` only.

**Cutover:** this tree moves with the kit into `modules/web/web/components/vendor/`
(same relative layout). Keep the `vendor/ui` segment stable so ignore rules, import
bans, and upstream diffs do not need a second rename.
