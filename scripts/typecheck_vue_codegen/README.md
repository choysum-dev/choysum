<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: LGPL-3.0-or-later
-->

# typecheck_vue_codegen

Node-only developer tool that regenerates Vue service-script **golden** files
used by `internal/typecheck` (`GoldenCoder`). Product binaries do **not** embed
this script (QuickJS language-core lands in a later PR).

## Requirements

- Node.js 22+
- `@vue/language-core@3.3.7` and a compatible `typescript` (repo root
  `node_modules` is enough when present)

Optional local install:

```bash
cd scripts/typecheck_vue_codegen
npm install
```

## Refresh goldens

From the repository root:

```bash
node scripts/typecheck_vue_codegen/refresh_golden.mjs
```

Writes:

- `internal/typecheck/testdata/vue/golden/<Fixture>.vue.service.txt`
- `internal/typecheck/testdata/vue/golden/<Fixture>.vue.mappings.json`

from `internal/typecheck/testdata/vue/fixtures/*.vue`.

## API

`create_service_script.mjs` exports `createServiceScript(fileName, source, options)`
matching the facade in `.dev/docs/infra/testing/typecheck_go_native_design.md` §5.1.
Helper `/// <reference>` paths are rewritten to:

- `/choysum-vue-virtual/types/template-helpers.d.ts`
- `/choysum-vue-virtual/types/props-fallback.d.ts`
