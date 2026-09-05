<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: LGPL-3.0-or-later
-->

# vuevirtual

QuickJS-embeddable facade over `@vue/language-core@3.3.7` that exports
`createServiceScript` (IIFE global `vuevirtual`). Used by
`internal/typecheck/vue.QuickJSCoder`. Product binaries load this via
`//go:embed`; runtime does **not** use Node.

## Generate (required before `go build` / tests that embed)

From the repository root:

```bash
# Prefer local install (large language-core + typescript; esm.sh often fails):
cd pkg/jsengine/scripts/vuevirtual && npm install && cd -
go generate ./pkg/jsengine/scripts/vuevirtual/...
```

Writes git-ignored `dist/index.js` (~3.9 MB). When `node_modules` is absent,
`gen.go` falls back to the esm.sh resolver (may fail for this package).

Pinned in `package.json` (and used by local npm install):

- `@vue/language-core@3.3.7`
- `typescript@6.0.3`
- `path-browserify@1.0.1`

## Do not reuse `vuesfc`

`vuesfc` is `@vue/compiler-sfc` runtime compile. This package is language-core
**typecheck** service-script codegen only.
