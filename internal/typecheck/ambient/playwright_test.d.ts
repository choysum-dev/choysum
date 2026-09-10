// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * Loose ambient for legacy Playwright e2e specs still partitioned onto Node.
 * Kept intentionally `any` so Go-native typecheck stays zero-Node until those
 * specs migrate fully to @choysum/e2e.
 */
declare module '@playwright/test' {
  export const test: any;
  export const expect: any;
  export type Page = any;
  export type Locator = any;
  export type Browser = any;
  export type BrowserContext = any;
  export type APIRequestContext = any;
  export type TestInfo = any;
}
