// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

import crypto from 'node:crypto'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

// Guardrail for IDE / direct `vitest` runs without choysum's generated --config.
// Vitest defaults to node_modules/.vite/vitest/<hash>/results.json; keep caches
// out of the repo. Official `choysum test --fe` still overrides cacheDir via its
// temporary config under .choysum/tmp/.../frontend/vite-cache/<app>.
//
// Do not add broad test.include here: bare discovery across the monorepo is not
// a supported runner entrypoint (use choysum test --fe instead).
//
// Hash the checkout path so shared /tmp hosts and parallel worktrees do not
// collide on a single cache directory.
const projectRoot = path.dirname(fileURLToPath(import.meta.url))
const checkoutHash = crypto.createHash('sha256').update(projectRoot).digest('hex').slice(0, 12)

export default {
  cacheDir: path.join(os.tmpdir(), `choysum-vite-${checkoutHash}`),
}
