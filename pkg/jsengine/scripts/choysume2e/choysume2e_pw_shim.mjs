// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * Playwright-compatible ESM shim for `@choysum/e2e` while dual-running PW specs
 * that still import shared utils (login/grpcweb) from this package name.
 *
 * Must be ESM so `import { expect } from '@choysum/e2e'` gets named exports
 * (a CJS `module.exports = require(...)` shim does not).
 */
export * from '@playwright/test';
