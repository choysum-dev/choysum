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

From the repository root (same path as `vuesfc` — esm.sh + local cache, no
`node_modules`):

```bash
go generate ./pkg/jsengine/scripts/vuevirtual/...
```

Writes git-ignored `dist/index.js` (several MB). Needs network on first run;
cache lives under `$CHOYSUM_HOME/pkg/esm` (default `~/.choysum/pkg/esm`).

`gen.go` pins versions via `esm.lock` and requests `?target=es2020&bundle`
so esm.sh can serve `@vue/language-core` (bare `?target=` alone may 500).

Pinned in `package.json` / `esm.lock`:

- `@vue/language-core@3.3.7`
- `typescript@6.0.3`
- `path-browserify@1.0.1`

## Do not reuse `vuesfc`

`vuesfc` is `@vue/compiler-sfc` runtime compile. This package is language-core
**typecheck** service-script codegen only.
